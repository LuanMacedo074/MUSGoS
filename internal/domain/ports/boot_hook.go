package ports

// BootHookEnv is what a boot hook may use: the configured scripts path (empty
// when scripts are disabled), the query builder and the logger.
type BootHookEnv struct {
	ScriptsPath  string
	QueryBuilder QueryBuilder
	Logger       Logger
}

// BootHook is game code the server runs once at boot, after the migrations,
// and again on every console "reload". Use it for state derived from the game's
// own files, such as publishing a catalog of the scripts on disk. A hook must be
// idempotent because it runs more than once per process.
type BootHook struct {
	Name string
	Run  func(env BootHookEnv) error
}
