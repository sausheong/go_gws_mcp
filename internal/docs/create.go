package docs

import (
	"context"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	docsapi "google.golang.org/api/docs/v1"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

// CreateArgs is the arg shape for create_doc.
type CreateArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
	Title           string `json:"title"`
}

// CreateDoc creates an empty Google Doc with the given title.
func CreateDoc(ctx context.Context, svc *docsapi.Service, userEmail string, a CreateArgs) (string, error) {
	if a.Title == "" {
		return "", errors.New("title is required")
	}
	created, err := svc.Documents.Create(&docsapi.Document{Title: a.Title}).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	link := fmt.Sprintf("https://docs.google.com/document/d/%s/edit", created.DocumentId)
	return fmt.Sprintf("Created doc %q (id: %s) — %s", created.Title, created.DocumentId, link), nil
}

func registerCreate(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("create_doc",
		mcp.WithDescription("Creates a new, empty Google Doc with the given title."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Document title")),
	)
	scopes := []string{auth.DocsScope}
	reg.Record("create_doc", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("create_doc", "Docs", scopes, docsFactory, c, email, CreateDoc))
}
