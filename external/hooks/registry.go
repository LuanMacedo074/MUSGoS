package hooks

import "fsos-server/internal/domain/ports"

// All is the game's boot hooks. Mirrors the migrations and queues registries:
// the engine runs them at boot and on the console "reload" command, game code
// registers concrete entries from an init() in this package.
var All []ports.BootHook

func Register(h ports.BootHook) {
	All = append(All, h)
}
