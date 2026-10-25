package inbound_test

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound"
	"fsos-server/internal/domain/ports"
)

// runConsole feeds input to a console and returns what it printed. The console
// writes to stdout, so it is captured through a pipe for the duration of Run.
func runConsole(t *testing.T, c *inbound.Console) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = stdout }()

	c.Run()

	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func newConsole(input string) *inbound.Console {
	return inbound.NewConsole(nil, &testutil.MockLogger{}, strings.NewReader(input), 0)
}

func TestConsole_ReloadReRunsTheBootHooksWithTheirEnv(t *testing.T) {
	var got []string
	hooks := []ports.BootHook{
		{Name: "catalog", Run: func(env ports.BootHookEnv) error { got = append(got, env.ScriptsPath); return nil }},
	}
	c := newConsole("reload\nreload\n").WithBootHooks(hooks, ports.BootHookEnv{ScriptsPath: "/srv/scripts"})

	out := runConsole(t, c)

	if len(got) != 2 || got[0] != "/srv/scripts" || got[1] != "/srv/scripts" {
		t.Fatalf("hook runs = %v, want two runs with the console's env", got)
	}
	if !strings.Contains(out, "Ran 1 boot hook(s).") {
		t.Fatalf("output = %q, want a reload confirmation", out)
	}
}

func TestConsole_ReloadReportsAFailingHook(t *testing.T) {
	hooks := []ports.BootHook{
		{Name: "catalog", Run: func(ports.BootHookEnv) error { return errors.New("table missing") }},
	}
	c := newConsole("reload\n").WithBootHooks(hooks, ports.BootHookEnv{})

	out := runConsole(t, c)

	if !strings.Contains(out, `Error reloading: boot hook "catalog": table missing`) {
		t.Fatalf("output = %q, want the hook's error", out)
	}
}

func TestConsole_ReloadWithoutHooksSaysSo(t *testing.T) {
	out := runConsole(t, newConsole("reload\n"))

	if !strings.Contains(out, "No boot hooks registered.") {
		t.Fatalf("output = %q, want the no-hooks message", out)
	}
}

func TestConsole_HelpListsReload(t *testing.T) {
	out := runConsole(t, newConsole("help\n"))

	if !strings.Contains(out, "  reload ") {
		t.Fatalf("help = %q, want a reload entry", out)
	}
}
