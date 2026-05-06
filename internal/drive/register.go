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

// RegisterTools wires Drive tools onto srv. Filled in across Tasks 6–11.
func RegisterTools(srv *server.MCPServer, registry *core.Registry, oauthClient *auth.OAuthClient, defaultEmail string) {
	// register* calls added in subsequent tasks.
	_ = srv
	_ = registry
	_ = oauthClient
	_ = defaultEmail
}
