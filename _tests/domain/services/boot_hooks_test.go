package services_test

import (
	"errors"
	"strings"
	"testing"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/services"
)

func TestRunBootHooks_RunsEveryHookInOrderWithTheEnv(t *testing.T) {
	env := ports.BootHookEnv{ScriptsPath: "/srv/scripts"}
	var ran []string
	hooks := []ports.BootHook{
		{Name: "first", Run: func(e ports.BootHookEnv) error { ran = append(ran, "first:"+e.ScriptsPath); return nil }},
		{Name: "second", Run: func(e ports.BootHookEnv) error { ran = append(ran, "second:"+e.ScriptsPath); return nil }},
	}

	if err := services.RunBootHooks(hooks, env); err != nil {
		t.Fatalf("RunBootHooks: %v", err)
	}

	want := []string{"first:/srv/scripts", "second:/srv/scripts"}
	if strings.Join(ran, ",") != strings.Join(want, ",") {
		t.Fatalf("ran = %v, want %v", ran, want)
	}
}

func TestRunBootHooks_FailureDoesNotStopLaterHooks(t *testing.T) {
	boom := errors.New("boom")
	secondRan := false
	hooks := []ports.BootHook{
		{Name: "broken", Run: func(ports.BootHookEnv) error { return boom }},
		{Name: "healthy", Run: func(ports.BootHookEnv) error { secondRan = true; return nil }},
	}

	err := services.RunBootHooks(hooks, ports.BootHookEnv{})

	if !secondRan {
		t.Fatal("a failing hook stopped the hooks after it")
	}
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want it to wrap the hook's error", err)
	}
	if !strings.Contains(err.Error(), `"broken"`) {
		t.Fatalf("err = %q, want it to name the failing hook", err)
	}
}

func TestRunBootHooks_NoHooksIsANoOp(t *testing.T) {
	if err := services.RunBootHooks(nil, ports.BootHookEnv{}); err != nil {
		t.Fatalf("RunBootHooks(nil): %v", err)
	}
}
