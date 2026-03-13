package outbound

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"

	lua "github.com/yuin/gopher-lua"
)

type LuaScriptEngine struct {
	scriptsDir    string
	logger        ports.Logger
	scriptTimeout time.Duration
	publisher     ports.QueuePublisher
}

func NewLuaScriptEngine(scriptsDir string, logger ports.Logger, scriptTimeoutSeconds int, publisher ports.QueuePublisher) *LuaScriptEngine {
	return &LuaScriptEngine{
		scriptsDir:    scriptsDir,
		logger:        logger,
		scriptTimeout: time.Duration(scriptTimeoutSeconds) * time.Second,
		publisher:     publisher,
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

	L.SetGlobal("mus", musMod)

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
