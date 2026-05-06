// Package sheets implements Google Sheets MCP tools.
package sheets

import (
	"context"

	"github.com/mark3labs/mcp-go/server"
	driveapi "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	sheetsapi "google.golang.org/api/sheets/v4"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

func sheetsFactory(ctx context.Context, opts ...option.ClientOption) (*sheetsapi.Service, error) {
	return sheetsapi.NewService(ctx, opts...)
}

// driveFactory is a local copy used by tools (list_spreadsheets) that need the
// Drive API. Kept local to avoid coupling internal/sheets to internal/drive.
func driveFactory(ctx context.Context, opts ...option.ClientOption) (*driveapi.Service, error) {
	return driveapi.NewService(ctx, opts...)
}

// RegisterTools wires Sheets tools onto srv. Filled in across Tasks 5–10.
func RegisterTools(srv *server.MCPServer, registry *core.Registry, oauthClient *auth.OAuthClient, defaultEmail string) {
	// register* calls added in subsequent tasks.
	_ = srv
	_ = registry
	_ = oauthClient
	_ = defaultEmail
}
