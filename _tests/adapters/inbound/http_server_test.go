package inbound_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound"
	"fsos-server/internal/adapters/outbound"
)

// httpScriptEngine builds a script engine over a temp dir with no sender and no
// session store — the exact shape the HTTP server runs with, so a script
// genuinely has no MUS session.
func httpScriptEngine(t *testing.T, scripts map[string]string) *outbound.LuaScriptEngine {
	t.Helper()
	dir := t.TempDir()
	for name, body := range scripts {
		path := filepath.Join(dir, name+".lua")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatalf("write script %s: %v", name, err)
		}
	}
	return outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil)
}

func newHTTPTestServer(t *testing.T, scripts map[string]string) *httptest.Server {
	t.Helper()
	engine := httpScriptEngine(t, scripts)
	adapter := inbound.NewHTTPServer("0", "127.0.0.1", engine, &testutil.MockLogger{})
	srv := httptest.NewServer(adapter.Handler())
	t.Cleanup(srv.Close)
	return srv
}

func decodeBody(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("response not JSON (%q): %v", string(raw), err)
	}
	return out
}

func TestHTTPServer_PingReturnsStructuredJSON(t *testing.T) {
	srv := newHTTPTestServer(t, map[string]string{
		"ping": `
			local req = mus.json.decode(mus.getContent())
			mus.response(mus.json.encode({
				status = 200,
				body = { pong = true, method = req.method, path = req.path },
			}))
		`,
	})

	resp, err := http.Get(srv.URL + "/ping")
	if err != nil {
		t.Fatalf("GET /ping: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	body := decodeBody(t, resp)
	if body["pong"] != true {
		t.Errorf("pong = %v, want true", body["pong"])
	}
	if body["method"] != "GET" {
		t.Errorf("method = %v, want GET", body["method"])
	}
	if body["path"] != "/ping" {
		t.Errorf("path = %v, want /ping", body["path"])
	}
}

func TestHTTPServer_PostBodyReachesScript(t *testing.T) {
	srv := newHTTPTestServer(t, map[string]string{
		"echo": `
			local req = mus.json.decode(mus.getContent())
			mus.response(mus.json.encode({ status = 201, body = req.body }))
		`,
	})

	resp, err := http.Post(srv.URL+"/echo", "application/json", strings.NewReader(`{"email":"a@b.com","n":7}`))
	if err != nil {
		t.Fatalf("POST /echo: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	if body["email"] != "a@b.com" {
		t.Errorf("email = %v, want a@b.com", body["email"])
	}
	if body["n"] != float64(7) {
		t.Errorf("n = %v, want 7", body["n"])
	}
}

func TestHTTPServer_NoMUSSession(t *testing.T) {
	// The adapter runs scripts with no sender and no session store, so a script
	// sees an empty sender and no mus.server module — proof no MUS session exists.
	srv := newHTTPTestServer(t, map[string]string{
		"probe": `
			mus.response(mus.json.encode({
				status = 200,
				body = { sender = mus.getSender(), has_server = mus.server ~= nil },
			}))
		`,
	})

	resp, err := http.Get(srv.URL + "/probe")
	if err != nil {
		t.Fatalf("GET /probe: %v", err)
	}
	body := decodeBody(t, resp)
	if body["sender"] != "" {
		t.Errorf("sender = %v, want empty", body["sender"])
	}
	if body["has_server"] != false {
		t.Errorf("has_server = %v, want false", body["has_server"])
	}
}

func TestHTTPServer_UnknownScriptIs404(t *testing.T) {
	srv := newHTTPTestServer(t, map[string]string{})
	resp, err := http.Get(srv.URL + "/nope")
	if err != nil {
		t.Fatalf("GET /nope: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestHTTPServer_InvalidJSONBodyIs400(t *testing.T) {
	srv := newHTTPTestServer(t, map[string]string{
		"echo": `mus.response(mus.json.encode({ status = 200, body = {} }))`,
	})
	resp, err := http.Post(srv.URL+"/echo", "application/json", strings.NewReader(`{not json`))
	if err != nil {
		t.Fatalf("POST /echo: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestHTTPServer_UnknownPathBeats400(t *testing.T) {
	// A malformed body sent to a nonexistent path must still resolve as 404 —
	// path existence is checked before the body is parsed.
	srv := newHTTPTestServer(t, map[string]string{})
	resp, err := http.Post(srv.URL+"/nope", "application/json", strings.NewReader(`{not json`))
	if err != nil {
		t.Fatalf("POST /nope: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (path resolves before body)", resp.StatusCode)
	}
}

func TestHTTPServer_ScriptWithoutResponseIs500(t *testing.T) {
	srv := newHTTPTestServer(t, map[string]string{
		"silent": `local x = 1 + 1`,
	})
	resp, err := http.Get(srv.URL + "/silent")
	if err != nil {
		t.Fatalf("GET /silent: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}

func TestHTTPServer_ScriptErrorIs500(t *testing.T) {
	srv := newHTTPTestServer(t, map[string]string{
		"boom": `error("kaboom")`,
	})
	resp, err := http.Get(srv.URL + "/boom")
	if err != nil {
		t.Fatalf("GET /boom: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}

func TestHTTPServer_RootPathIs404(t *testing.T) {
	srv := newHTTPTestServer(t, map[string]string{
		"ping": `mus.response(mus.json.encode({ status = 200, body = {} }))`,
	})
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestHTTPServer_DefaultStatusIs200(t *testing.T) {
	// A response envelope that omits status defaults to 200.
	srv := newHTTPTestServer(t, map[string]string{
		"nostatus": `mus.response(mus.json.encode({ body = { ok = true } }))`,
	})
	resp, err := http.Get(srv.URL + "/nostatus")
	if err != nil {
		t.Fatalf("GET /nostatus: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}
