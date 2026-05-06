package auth

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// CallbackServer runs a minimal HTTP server with /oauth2callback for stdio mode.
// In streamable-HTTP mode the main HTTP mux handles the route directly.
type CallbackServer struct {
	port   int
	host   string
	client *OAuthClient

	srv     *http.Server
	mu      sync.Mutex
	running bool
}

// NewCallbackServer constructs a callback server bound to host:port.
func NewCallbackServer(host string, port int, client *OAuthClient) *CallbackServer {
	return &CallbackServer{host: host, port: port, client: client}
}

// Start listens in a goroutine and returns when the server is reachable, or error.
func (s *CallbackServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle("/oauth2callback", s.HTTPHandler())

	s.srv = &http.Server{
		Addr:    net.JoinHostPort(s.host, strconv.Itoa(s.port)),
		Handler: mux,
	}

	listener, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		return fmt.Errorf("bind callback server: %w", err)
	}

	go func() {
		if err := s.srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("callback server error", "err", err)
		}
	}()
	s.running = true
	slog.Info("OAuth callback server listening", "addr", s.srv.Addr)
	return nil
}

// Stop gracefully shuts down the server.
func (s *CallbackServer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running || s.srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.srv.Shutdown(ctx)
	s.running = false
}

// HTTPHandler is the OAuth callback HTTP handler; useful for embedding into the
// main HTTP mux in streamable-HTTP mode (instead of running a separate server).
func (s *CallbackServer) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if errParam := q.Get("error"); errParam != "" {
			renderError(w, fmt.Sprintf("Google returned error: %s", errParam))
			return
		}
		state := q.Get("state")
		code := q.Get("code")
		if state == "" || code == "" {
			renderError(w, "Missing state or code in callback")
			return
		}

		email, err := s.client.HandleAuthCallback(r.Context(), state, code)
		if err != nil {
			renderError(w, err.Error())
			return
		}
		renderSuccess(w, email)
	}
}

func renderSuccess(w http.ResponseWriter, email string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>Authentication Successful</title></head>
<body style="font-family:system-ui;max-width:480px;margin:64px auto;padding:24px;background:#f4f4f5">
<h2>✅ Authentication successful</h2>
<p>Authenticated as <strong>%s</strong>.</p>
<p>You can close this tab and return to your assistant.</p>
</body></html>`, html.EscapeString(email))
}

func renderError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>Authentication Failed</title></head>
<body style="font-family:system-ui;max-width:480px;margin:64px auto;padding:24px;background:#fee">
<h2>❌ Authentication failed</h2>
<pre style="white-space:pre-wrap">%s</pre>
</body></html>`, html.EscapeString(msg))
}
