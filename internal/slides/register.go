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

// RegisterTools wires all Slides tools onto srv and records them in registry.
func RegisterTools(srv *server.MCPServer, registry *core.Registry, oauthClient *auth.OAuthClient, defaultEmail string) {
	registerCreate(srv, registry, oauthClient, defaultEmail)
	registerGet(srv, registry, oauthClient, defaultEmail)
	registerBatchUpdate(srv, registry, oauthClient, defaultEmail)
	registerGetPage(srv, registry, oauthClient, defaultEmail)
	registerGetThumbnail(srv, registry, oauthClient, defaultEmail)
}
