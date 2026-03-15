package outbound

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"

	"github.com/google/uuid"
	lua "github.com/yuin/gopher-lua"
)

type LuaScriptEngine struct {
	scriptsDir    string
	logger        ports.Logger
	scriptTimeout time.Duration
	publisher     ports.QueuePublisher
	sender        ports.MessageSender
	db            ports.DBAdapter
	queryBuilder  ports.QueryBuilder
	sessionStore  ports.SessionStore
	cache         ports.Cache
}

func NewLuaScriptEngine(scriptsDir string, logger ports.Logger, scriptTimeoutSeconds int, publisher ports.QueuePublisher, sender ports.MessageSender, db ports.DBAdapter, queryBuilder ports.QueryBuilder, sessionStore ports.SessionStore, cache ports.Cache) *LuaScriptEngine {
	return &LuaScriptEngine{
		scriptsDir:    scriptsDir,
		logger:        logger,
		scriptTimeout: time.Duration(scriptTimeoutSeconds) * time.Second,
		publisher:     publisher,
		sender:        sender,
		db:            db,
		queryBuilder:  queryBuilder,
		sessionStore:  sessionStore,
		cache:         cache,
	}
}

func (e *LuaScriptEngine) HasScript(subject string) bool {
	path := filepath.Join(e.scriptsDir, subject+".lua")
	_, err := os.Stat(path)
	return err == nil
}

func (e *LuaScriptEngine) Execute(msg *ports.ScriptMessage) (*ports.ScriptResult, error) {
	path := filepath.Join(e.scriptsDir, msg.Subject+".lua")

	// Fresh VM per execution — intentionally not pooled.
	// This ensures thread safety (no shared state between concurrent calls)
	// at the cost of VM creation overhead. If throughput becomes an issue,
	// consider a sync.Pool of pre-configured LStates.
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer L.Close()

	// Execution timeout to prevent runaway scripts (e.g. infinite loops)
	ctx, cancel := context.WithTimeout(context.Background(), e.scriptTimeout)
	defer cancel()
	L.SetContext(ctx)

	// Open only safe libs — no os, io, debug, or package (which exposes loadlib/filesystem)
	for _, pair := range []struct {
		name string
		fn   lua.LGFunction
	}{
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
	} {
		if err := L.CallByParam(lua.P{
			Fn:      L.NewFunction(pair.fn),
			NRet:    0,
			Protect: true,
		}, lua.LString(pair.name)); err != nil {
			return nil, fmt.Errorf("failed to open lib %s: %w", pair.name, err)
		}
	}

	// Build the mus module
	var result *ports.ScriptResult

	musMod := L.NewTable()

	musMod.RawSetString("getSender", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(msg.SenderID))
		return 1
	}))

	musMod.RawSetString("getContent", L.NewFunction(func(L *lua.LState) int {
		L.Push(lingo.LValueToLua(L, msg.Content))
		return 1
	}))

	musMod.RawSetString("response", L.NewFunction(func(L *lua.LState) int {
		arg := L.Get(1)
		content := lingo.LuaToLValue(arg)
		result = &ports.ScriptResult{Content: content}
		L.Push(arg)
		return 1
	}))

	musMod.RawSetString("publish", L.NewFunction(func(L *lua.LState) int {
		topic := L.CheckString(1)
		content := L.Get(2)
		payload := lingo.LuaToLValue(content).GetBytes()
		if e.publisher != nil {
			if err := e.publisher.Publish(topic, payload); err != nil {
				e.logger.Error("mus.publish failed", map[string]interface{}{
					"topic": topic,
					"error": err.Error(),
				})
			}
		}
		return 0
	}))

	musMod.RawSetString("sendMessage", L.NewFunction(func(L *lua.LState) int {
		recipientID := L.CheckString(1)
		if recipientID == "" {
			L.ArgError(1, "recipientID must not be empty")
			return 0
		}
		subject := L.CheckString(2)
		content := L.Get(3)
		lingoContent := lingo.LuaToLValue(content)
		if e.sender != nil {
			if err := e.sender.SendMessage(msg.SenderID, recipientID, subject, lingoContent); err != nil {
				e.logger.Error("mus.sendMessage failed", map[string]interface{}{
					"recipientID": recipientID,
					"subject":     subject,
					"error":       err.Error(),
				})
			}
		}
		return 0
	}))

	// Register mus.db module (query builder + standard DB operations)
	if e.db != nil || e.queryBuilder != nil {
		registerDBModule(L, musMod, e.db, e.queryBuilder, e.logger)
	}

	// Register mus.server module
	if e.sessionStore != nil {
		registerServerModule(L, musMod, e.sessionStore)
	}

	// Register mus.cache module
	if e.cache != nil {
		registerCacheModule(L, musMod, e.cache)
	}

	// Register mus.log module
	registerLogModule(L, musMod, e.logger)

	// Register mus.time module
	registerTimeModule(L, musMod)

	// Register mus.json module
	registerJsonModule(L, musMod)

	musMod.RawSetString("uuid", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(uuid.New().String()))
		return 1
	}))

	L.SetGlobal("mus", musMod)

	// Sandboxed require: only loads .lua files from scriptsDir/lib/
	loaded := L.NewTable()
	L.SetGlobal("require", L.NewFunction(func(L *lua.LState) int {
		modName := L.CheckString(1)

		// Return cached module if already loaded
		if cached := loaded.RawGetString(modName); cached != lua.LNil {
			L.Push(cached)
			return 1
		}

		// Resolve path — only allow lib/ prefix, no ".." traversal
		if filepath.IsAbs(modName) || containsDotDot(modName) {
			L.ArgError(1, "invalid module name: "+modName)
			return 0
		}

		modPath := filepath.Join(e.scriptsDir, modName+".lua")

		// Security: ensure resolved path stays within scriptsDir
		absModPath, err := filepath.Abs(modPath)
		if err != nil {
			L.ArgError(1, "cannot resolve module path: "+modName)
			return 0
		}
		absScriptsDir, _ := filepath.Abs(e.scriptsDir)
		if !isSubpath(absScriptsDir, absModPath) {
			L.ArgError(1, "module path escapes scripts directory: "+modName)
			return 0
		}

		if _, err := os.Stat(modPath); err != nil {
			L.ArgError(1, "module not found: "+modName)
			return 0
		}

		// Load and execute the module file
		fn, err := L.LoadFile(modPath)
		if err != nil {
			L.RaiseError("failed to load module %q: %s", modName, err.Error())
			return 0
		}

		L.Push(fn)
		if err := L.PCall(0, 1, nil); err != nil {
			L.RaiseError("failed to execute module %q: %s", modName, err.Error())
			return 0
		}

		ret := L.Get(-1)
		if ret == lua.LNil {
			ret = lua.LTrue
		}

		// Cache the result
		loaded.RawSetString(modName, ret)

		L.Push(ret)
		return 1
	}))

	// Sandboxed dofile: resolves paths relative to scriptsDir, same security as require
	L.SetGlobal("dofile", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)

		if filepath.IsAbs(name) || containsDotDot(name) {
			L.ArgError(1, "invalid path: "+name)
			return 0
		}

		filePath := filepath.Join(e.scriptsDir, name)

		absFilePath, err := filepath.Abs(filePath)
		if err != nil {
			L.ArgError(1, "cannot resolve path: "+name)
			return 0
		}
		absScriptsDir, _ := filepath.Abs(e.scriptsDir)
		if !isSubpath(absScriptsDir, absFilePath) {
			L.ArgError(1, "path escapes scripts directory: "+name)
			return 0
		}

		if err := L.DoFile(filePath); err != nil {
			L.RaiseError("dofile %q failed: %s", name, err.Error())
			return 0
		}
		return 0
	}))

	if err := L.DoFile(path); err != nil {
		return nil, fmt.Errorf("script %q execution failed: %w", msg.Subject, err)
	}

	// Scripts must use mus.response() to produce a result.
	// If they don't, the result is LVoid (no response).
	if result == nil {
		result = &ports.ScriptResult{Content: lingo.NewLVoid()}
	}

	return result, nil
}

func containsDotDot(path string) bool {
	return strings.Contains(path, "..")
}

func isSubpath(parent, child string) bool {
	parent = filepath.Clean(parent) + string(filepath.Separator)
	child = filepath.Clean(child)
	return strings.HasPrefix(child, parent)
}
