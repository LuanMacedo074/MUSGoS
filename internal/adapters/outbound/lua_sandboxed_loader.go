package outbound

import (
	"path/filepath"
	"regexp"

	lua "github.com/yuin/gopher-lua"
)

var sandboxedChunkID = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// registerSandboxedLoader exposes mus.loadSandboxed(id), which runs
// <scriptsDir>/<chunkDir>/<id>.lua in an environment holding only safe Lua
// utilities and returns the table the chunk returns. The chunk is restricted
// before it executes: changing its environment afterwards would leave
// module-level writes and captured privileged upvalues unrestricted.
func registerSandboxedLoader(L *lua.LState, musMod *lua.LTable, scriptsDir, chunkDir string) {
	musMod.RawSetString("loadSandboxed", L.NewFunction(func(L *lua.LState) int {
		id := L.CheckString(1)
		if !sandboxedChunkID.MatchString(id) {
			L.ArgError(1, "invalid sandboxed chunk id")
			return 0
		}
		// Containment is checked on symlink-resolved paths: chunkDir must sit
		// strictly inside scriptsDir (never the root itself), and the chunk file
		// must resolve inside chunkDir.
		realScriptsDir, err := resolveRealPath(scriptsDir)
		if err != nil {
			L.RaiseError("load sandboxed chunk %q: %s", id, err)
			return 0
		}
		realChunkDir, err := resolveRealPath(filepath.Join(scriptsDir, chunkDir))
		if err != nil {
			L.RaiseError("load sandboxed chunk %q: %s", id, err)
			return 0
		}
		if !isSubpath(realScriptsDir, realChunkDir) {
			L.RaiseError("load sandboxed chunk %q: directory is outside the scripts directory", id)
			return 0
		}
		realChunk, err := resolveRealPath(filepath.Join(realChunkDir, id+".lua"))
		if err != nil {
			L.RaiseError("load sandboxed chunk %q: %s", id, err)
			return 0
		}
		if !isSubpath(realChunkDir, realChunk) {
			L.RaiseError("load sandboxed chunk %q: file is outside the sandboxed directory", id)
			return 0
		}
		fn, err := L.LoadFile(realChunk)
		if err != nil {
			L.RaiseError("load sandboxed chunk %q: %s", id, err)
			return 0
		}
		env := L.NewTable()
		for _, name := range []string{
			"assert", "error", "ipairs", "next", "pairs", "pcall", "select",
			"tonumber", "tostring", "type", "unpack", "xpcall",
		} {
			env.RawSetString(name, L.GetGlobal(name))
		}
		for _, name := range []string{"math", "string", "table"} {
			library, ok := L.GetGlobal(name).(*lua.LTable)
			if !ok {
				L.RaiseError("sandbox standard library %q is unavailable", name)
				return 0
			}
			clone := L.NewTable()
			library.ForEach(func(key, value lua.LValue) {
				clone.RawSet(key, value)
			})
			env.RawSetString(name, clone)
		}
		fn.Env = env
		L.Push(fn)
		if err := L.PCall(0, 1, nil); err != nil {
			L.RaiseError("execute sandboxed chunk %q: %s", id, err)
			return 0
		}
		if L.Get(-1).Type() != lua.LTTable {
			L.RaiseError("sandboxed chunk %q must return a table", id)
			return 0
		}
		return 1
	}))
}

// resolveRealPath returns the absolute path with every symlink resolved, so a
// containment check sees where the file actually lives.
func resolveRealPath(path string) (string, error) {
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	return filepath.Abs(real)
}
