// Package docs implements Google Docs MCP tools.
package docs

import (
	"context"

	"github.com/mark3labs/mcp-go/server"
	docsapi "google.golang.org/api/docs/v1"
	driveapi "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

func docsFactory(ctx context.Context, opts ...option.ClientOption) (*docsapi.Service, error) {
	return docsapi.NewService(ctx, opts...)
}

// driveFactory is a local copy used by tools (search_docs, get_doc_as_markdown)
// that need the Drive API. Kept local to avoid coupling internal/docs to
// internal/drive.
func driveFactory(ctx context.Context, opts ...option.ClientOption) (*driveapi.Service, error) {
	return driveapi.NewService(ctx, opts...)
}

// RegisterTools wires Docs tools onto srv. Filled in across Tasks 5–10.
func RegisterTools(srv *server.MCPServer, registry *core.Registry, oauthClient *auth.OAuthClient, defaultEmail string) {
	// register* calls added in subsequent tasks.
	_ = srv
	_ = registry
	_ = oauthClient
	_ = defaultEmail
}
