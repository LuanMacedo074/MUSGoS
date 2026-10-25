package services

import (
	"errors"
	"fmt"

	"fsos-server/internal/domain/ports"
)

// RunBootHooks runs every hook in order with the same env. A failing hook does
// not stop the ones after it; the returned error joins every failure, each
// prefixed with the hook's name.
func RunBootHooks(hooks []ports.BootHook, env ports.BootHookEnv) error {
	var errs []error
	for _, h := range hooks {
		if err := h.Run(env); err != nil {
			errs = append(errs, fmt.Errorf("boot hook %q: %w", h.Name, err))
		}
	}
	return errors.Join(errs...)
}
