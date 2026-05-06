package drive

import (
	"context"

	"github.com/mark3labs/mcp-go/server"
	driveapi "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

func driveFactory(ctx context.Context, opts ...option.ClientOption) (*driveapi.Service, error) {
	return driveapi.NewService(ctx, opts...)
}

// RegisterTools wires all Drive tools onto srv and records them in registry.
func RegisterTools(srv *server.MCPServer, registry *core.Registry, oauthClient *auth.OAuthClient, defaultEmail string) {
	registerSearch(srv, registry, oauthClient, defaultEmail)
	registerList(srv, registry, oauthClient, defaultEmail)
	registerGetContent(srv, registry, oauthClient, defaultEmail)
	registerCreateFolder(srv, registry, oauthClient, defaultEmail)
	registerCreateFile(srv, registry, oauthClient, defaultEmail)
}
