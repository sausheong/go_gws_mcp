package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/mark3labs/mcp-go/server"

	"github.com/sausheong/go_gws_mcp/internal/auth"
)

// Transport names accepted by Run.
const (
	TransportStdio          = "stdio"
	TransportStreamableHTTP = "streamable-http"
)

// Run starts the MCP server on the configured transport and blocks until exit.
// In stdio mode, also starts the minimal OAuth callback HTTP server on
// cfg.Host:cfg.Port for the OAuth flow. In streamable-HTTP mode, the callback
// is mounted on the same HTTP mux as the MCP endpoint.
func Run(ctx context.Context, srv *server.MCPServer, cfg *Config, oauthClient *auth.OAuthClient) error {
	switch cfg.Transport {
	case TransportStdio:
		callback := auth.NewCallbackServer(callbackHost(cfg), cfg.Port, oauthClient)
		if err := callback.Start(); err != nil {
			return err
		}
		defer callback.Stop()
		return server.ServeStdio(srv)

	case TransportStreamableHTTP:
		mux := http.NewServeMux()
		mux.HandleFunc("/health", healthHandler(cfg))
		callback := auth.NewCallbackServer(cfg.Host, cfg.Port, oauthClient)
		mux.Handle("/oauth2callback", callback.HTTPHandler())

		streamable := server.NewStreamableHTTPServer(srv)
		mux.Handle("/mcp", streamable)
		mux.Handle("/mcp/", streamable)

		addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
		httpSrv := &http.Server{Addr: addr, Handler: mux}
		go func() {
			<-ctx.Done()
			_ = httpSrv.Close()
		}()
		return httpSrv.ListenAndServe()

	default:
		return fmt.Errorf("unknown transport: %s", cfg.Transport)
	}
}

// callbackHost returns the host the stdio callback server should bind to,
// extracted from cfg.BaseURI (e.g. "http://localhost" -> "localhost").
func callbackHost(cfg *Config) string {
	// Strip scheme.
	uri := cfg.BaseURI
	for _, prefix := range []string{"https://", "http://"} {
		if len(uri) > len(prefix) && uri[:len(prefix)] == prefix {
			return uri[len(prefix):]
		}
	}
	return "localhost"
}

func healthHandler(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":    "healthy",
			"service":   "go-gws-mcp",
			"transport": cfg.Transport,
		})
	}
}
