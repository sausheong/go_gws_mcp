package docs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	docsapi "google.golang.org/api/docs/v1"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

// GetContentArgs is the arg shape for get_doc_content.
type GetContentArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
	DocumentID      string `json:"document_id"`
}

// GetDocContent returns the doc title + a plain-text rendering of the body.
// Walks Body.Content, extracting TextRun content from paragraphs and table cells.
// Section breaks and TOCs are skipped (rendered as a blank line).
func GetDocContent(ctx context.Context, svc *docsapi.Service, userEmail string, a GetContentArgs) (string, error) {
	if a.DocumentID == "" {
		return "", errors.New("document_id is required")
	}
	doc, err := svc.Documents.Get(a.DocumentID).Context(ctx).Do()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Document: %s\nID: %s\n---\n", doc.Title, doc.DocumentId)

	if doc.Body != nil {
		for _, el := range doc.Body.Content {
			renderStructural(&b, el)
		}
	}
	return b.String(), nil
}

func renderStructural(b *strings.Builder, el *docsapi.StructuralElement) {
	if el == nil {
		return
	}
	switch {
	case el.Paragraph != nil:
		for _, pe := range el.Paragraph.Elements {
			if pe.TextRun != nil {
				b.WriteString(pe.TextRun.Content)
			}
		}
	case el.Table != nil:
		for _, row := range el.Table.TableRows {
			for i, cell := range row.TableCells {
				if i > 0 {
					b.WriteString(" | ")
				}
				for _, inner := range cell.Content {
					renderStructural(b, inner)
				}
			}
		}
	default:
		b.WriteString("\n")
	}
}

func registerGetContent(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("get_doc_content",
		mcp.WithDescription("Retrieves a Google Doc and returns its title + plain-text body (paragraphs and tables only; formatting and embeds are dropped)."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("document_id", mcp.Required(), mcp.Description("Google Doc ID")),
	)
	scopes := []string{auth.DocsReadonlyScope}
	reg.Record("get_doc_content", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("get_doc_content", "Docs", scopes, docsFactory, c, email, GetDocContent))
}
