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

// ModifyTextArgs is the arg shape for modify_doc_text.
type ModifyTextArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
	DocumentID      string `json:"document_id"`
	Text            string `json:"text"`
	Index           int64  `json:"index,omitempty"` // 0 = append at end of body
}

// ModifyDocText inserts text into a Doc. Index 0 (default) appends at end of body;
// any positive index inserts at that absolute position.
func ModifyDocText(ctx context.Context, svc *docsapi.Service, userEmail string, a ModifyTextArgs) (string, error) {
	if a.DocumentID == "" {
		return "", errors.New("document_id is required")
	}
	if a.Text == "" {
		return "", errors.New("text is required")
	}

	insert := &docsapi.InsertTextRequest{Text: a.Text}
	if a.Index > 0 {
		insert.Location = &docsapi.Location{Index: a.Index}
	} else {
		insert.EndOfSegmentLocation = &docsapi.EndOfSegmentLocation{}
	}

	req := &docsapi.BatchUpdateDocumentRequest{
		Requests: []*docsapi.Request{{InsertText: insert}},
	}

	if _, err := svc.Documents.BatchUpdate(a.DocumentID, req).Context(ctx).Do(); err != nil {
		return "", err
	}

	mode := "appended at end of body"
	if a.Index > 0 {
		mode = fmt.Sprintf("inserted at index %d", a.Index)
	}
	return fmt.Sprintf("Modified doc %s — %s (%d char(s) inserted)", a.DocumentID, mode, len(a.Text)), nil
}

func registerModifyText(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("modify_doc_text",
		mcp.WithDescription("Inserts text into a Google Doc. By default appends at end of body; pass index>0 to insert at an absolute position."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("document_id", mcp.Required(), mcp.Description("Google Doc ID")),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text to insert")),
		mcp.WithNumber("index", mcp.Description("Optional absolute insertion index; 0 (default) appends at end of body")),
	)
	scopes := []string{auth.DocsScope}
	reg.Record("modify_doc_text", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("modify_doc_text", "Docs", scopes, docsFactory, c, email, ModifyDocText))
}
