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
		// The id cannot leave chunkDir, so containing chunkDir strictly inside
		// scriptsDir (never the root itself) contains every chunk.
		absChunkDir, err := filepath.Abs(filepath.Join(scriptsDir, chunkDir))
		if err != nil {
			L.RaiseError("load sandboxed chunk %q: %s", id, err)
			return 0
		}
		absScriptsDir, _ := filepath.Abs(scriptsDir)
		if !isSubpath(absScriptsDir, absChunkDir) {
			L.RaiseError("load sandboxed chunk %q: directory is outside the scripts directory", id)
			return 0
		}
		fn, err := L.LoadFile(filepath.Join(absChunkDir, id+".lua"))
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
