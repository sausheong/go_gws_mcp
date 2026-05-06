package docs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	driveapi "google.golang.org/api/drive/v3"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

// MaxMarkdownBytes caps the inlined markdown body.
const MaxMarkdownBytes = 5 * 1024 * 1024 // 5 MiB

// GetMarkdownArgs is the arg shape for get_doc_as_markdown.
type GetMarkdownArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
	DocumentID      string `json:"document_id"`
}

// GetDocAsMarkdown exports a Google Doc as markdown via Drive's text/markdown
// export endpoint (added to Drive API in 2024).
func GetDocAsMarkdown(ctx context.Context, svc *driveapi.Service, userEmail string, a GetMarkdownArgs) (string, error) {
	if a.DocumentID == "" {
		return "", errors.New("document_id is required")
	}
	meta, err := svc.Files.Get(a.DocumentID).
		Fields("id, name, mimeType, webViewLink").
		Context(ctx).Do()
	if err != nil {
		return "", err
	}
	if meta.MimeType != "application/vnd.google-apps.document" {
		return "", fmt.Errorf("file %s is not a Google Doc (mimeType=%s)", a.DocumentID, meta.MimeType)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Document: %s\nID: %s\nLink: %s\n---\n", meta.Name, meta.Id, meta.WebViewLink)

	resp, err := svc.Files.Export(a.DocumentID, "text/markdown").Context(ctx).Download()
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxMarkdownBytes))
	if err != nil {
		return "", err
	}
	b.Write(body)
	return b.String(), nil
}

func registerGetMarkdown(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("get_doc_as_markdown",
		mcp.WithDescription("Exports a Google Doc as markdown using Drive's text/markdown export."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("document_id", mcp.Required(), mcp.Description("Google Doc ID")),
	)
	scopes := []string{auth.DriveReadonlyScope}
	reg.Record("get_doc_as_markdown", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("get_doc_as_markdown", "Docs", scopes, driveFactory, c, email, GetDocAsMarkdown))
}
