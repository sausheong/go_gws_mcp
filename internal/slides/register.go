// Package slides implements Google Slides MCP tools.
package slides

import (
	"context"

	"github.com/mark3labs/mcp-go/server"
	"google.golang.org/api/option"
	slidesapi "google.golang.org/api/slides/v1"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

func slidesFactory(ctx context.Context, opts ...option.ClientOption) (*slidesapi.Service, error) {
	return slidesapi.NewService(ctx, opts...)
}

// RegisterTools wires Slides tools onto srv. Filled in across Tasks 5–10.
func RegisterTools(srv *server.MCPServer, registry *core.Registry, oauthClient *auth.OAuthClient, defaultEmail string) {
	// register* calls added in subsequent tasks.
	_ = srv
	_ = registry
	_ = oauthClient
	_ = defaultEmail
}
