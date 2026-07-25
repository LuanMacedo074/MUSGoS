package inbound

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

// maxHTTPBodyBytes caps the request body the HTTP server will read. A cheap
// guard against a runaway upload; generous for the small JSON payloads scripts
// consume.
const maxHTTPBodyBytes = 1 << 20 // 1 MiB

// HTTPServer is an optional, generic HTTP inbound adapter. When enabled it maps
// a request path to a Lua script (external/https/<name>.lua), invokes it with
// the request as a JSON envelope and NO MUS session, and writes the script's
// structured result (status + JSON body) as the HTTP response.
//
// The engine stays agnostic to what the scripts do: it owns the pipe (this
// adapter) and the primitives (the mus.* modules on the script VM); the request
// semantics — auth, or anything else — live entirely in the scripts. Mirrors the
// hexagonal inbound-adapter pattern of the metrics/UDP servers; no change to the
// MUS TCP path.
type HTTPServer struct {
	port     string
	bindAddr string
	engine   ports.ScriptEngine
	logger   ports.Logger
	// mu guards server, written in Start() and read from Shutdown() on another goroutine.
	mu     sync.Mutex
	server *http.Server
}

func NewHTTPServer(port, bindAddr string, engine ports.ScriptEngine, logger ports.Logger) *HTTPServer {
	if bindAddr == "" {
		bindAddr = "127.0.0.1"
	}
	return &HTTPServer{
		port:     port,
		bindAddr: bindAddr,
		engine:   engine,
		logger:   logger,
	}
}

// Handler returns the HTTP handler that bridges requests to scripts. Exposed so
// the bridge can be exercised without binding a socket.
func (s *HTTPServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	return mux
}

func (s *HTTPServer) Start() error {
	addr := s.bindAddr + ":" + s.port
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	s.mu.Lock()
	s.server = srv
	s.mu.Unlock()

	s.logger.Info("HTTP server listening", map[string]interface{}{
		"address": addr,
	})

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server error: %w", err)
	}
	return nil
}

func (s *HTTPServer) Shutdown() {
	s.mu.Lock()
	srv := s.server
	s.mu.Unlock()
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}
}

// scriptResponse is the structured envelope a script returns via mus.response():
// an HTTP status plus a JSON body, re-emitted verbatim on the wire.
type scriptResponse struct {
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

func (s *HTTPServer) handle(w http.ResponseWriter, r *http.Request) {
	// The subject is the path with its surrounding slashes stripped; "/ping" ->
	// "ping". The engine enforces containment (no absolute/".." escapes). Resolve
	// the script's existence first so a bad path is a clean 404, before any
	// body parsing that could otherwise mask it as a 400.
	subject := strings.Trim(r.URL.Path, "/")
	if subject == "" || !s.engine.HasScript(subject) {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, maxHTTPBodyBytes))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "cannot read request body")
		return
	}

	var body interface{}
	if len(bytes.TrimSpace(bodyBytes)) > 0 {
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}

	envelope, err := json.Marshal(map[string]interface{}{
		"method": r.Method,
		"path":   r.URL.Path,
		"body":   body,
	})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "cannot encode request")
		return
	}

	// No sender, no session store on this engine -> no MUS session is established.
	result, err := s.engine.Execute(&ports.ScriptMessage{
		Subject:  subject,
		SenderID: "",
		Content:  lingo.NewLString(string(envelope)),
	})
	if err != nil {
		s.logger.Error("HTTP script failed", map[string]interface{}{
			"subject": subject,
			"error":   err.Error(),
		})
		s.writeError(w, http.StatusInternalServerError, "script error")
		return
	}

	str, ok := result.Content.(*lingo.LString)
	if !ok {
		// The script ran but never called mus.response() with a structured envelope.
		s.logger.Error("HTTP script produced no response", map[string]interface{}{
			"subject": subject,
		})
		s.writeError(w, http.StatusInternalServerError, "script produced no response")
		return
	}

	var resp scriptResponse
	if err := json.Unmarshal([]byte(str.Value), &resp); err != nil {
		s.logger.Error("HTTP script returned malformed envelope", map[string]interface{}{
			"subject": subject,
			"error":   err.Error(),
		})
		s.writeError(w, http.StatusInternalServerError, "malformed script response")
		return
	}

	status := resp.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if len(resp.Body) > 0 {
		w.Write(resp.Body)
	} else {
		w.Write([]byte("{}"))
	}
}

func (s *HTTPServer) writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
