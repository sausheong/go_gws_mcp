package docs

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	driveapi "google.golang.org/api/drive/v3"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

// SearchArgs is the arg shape for search_docs.
type SearchArgs struct {
	Query           string `json:"query,omitempty"` // optional extra Drive query terms
	UserGoogleEmail string `json:"user_google_email"`
	PageSize        int    `json:"page_size,omitempty"`
	PageToken       string `json:"page_token,omitempty"`
}

// SearchDocs lists Google Docs matching an optional extra query.
// The MIME-type filter is always applied; user query is AND-combined.
func SearchDocs(ctx context.Context, svc *driveapi.Service, userEmail string, a SearchArgs) (string, error) {
	pageSize := a.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	q := "mimeType = 'application/vnd.google-apps.document' and trashed = false"
	if strings.TrimSpace(a.Query) != "" {
		q = fmt.Sprintf("%s and (%s)", q, a.Query)
	}

	call := svc.Files.List().
		Q(q).
		PageSize(int64(pageSize)).
		Fields("nextPageToken, files(id, name, modifiedTime, webViewLink)").
		Spaces("drive")
	if a.PageToken != "" {
		call = call.PageToken(a.PageToken)
	}
	resp, err := call.Context(ctx).Do()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Docs search (extra query: %q, user: %s)\n", a.Query, userEmail)
	fmt.Fprintf(&b, "Found %d doc(s)\n\n", len(resp.Files))
	for _, f := range resp.Files {
		fmt.Fprintf(&b, "- %s\n  ID: %s | Modified: %s\n  Link: %s\n",
			f.Name, f.Id, f.ModifiedTime, f.WebViewLink)
	}
	if resp.NextPageToken != "" {
		fmt.Fprintf(&b, "\nNext page token: %s\n", resp.NextPageToken)
	}
	return b.String(), nil
}

func registerSearch(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("search_docs",
		mcp.WithDescription("Searches Google Docs in the user's Drive (filters by Doc MIME type). Optional extra Drive query is AND-combined."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("query", mcp.Description("Optional extra Drive query terms (e.g., \"name contains 'budget'\")")),
		mcp.WithNumber("page_size", mcp.Description("Max results, default 10, max 100")),
		mcp.WithString("page_token", mcp.Description("Pagination token for next page")),
	)
	scopes := []string{auth.DriveReadonlyScope}
	reg.Record("search_docs", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("search_docs", "Docs", scopes, driveFactory, c, email, SearchDocs))
}
