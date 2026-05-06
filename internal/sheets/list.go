package sheets

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

// ListArgs is the arg shape for list_spreadsheets.
type ListArgs struct {
	Query           string `json:"query,omitempty"` // optional extra Drive query terms
	UserGoogleEmail string `json:"user_google_email"`
	PageSize        int    `json:"page_size,omitempty"`
	PageToken       string `json:"page_token,omitempty"`
}

// ListSpreadsheets lists Google Sheets in the user's Drive (filters by Sheet MIME).
// Optional extra Drive query is AND-combined.
func ListSpreadsheets(ctx context.Context, svc *driveapi.Service, userEmail string, a ListArgs) (string, error) {
	pageSize := a.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	q := "mimeType = 'application/vnd.google-apps.spreadsheet' and trashed = false"
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
	fmt.Fprintf(&b, "Spreadsheets (extra query: %q, user: %s)\n", a.Query, userEmail)
	fmt.Fprintf(&b, "Found %d spreadsheet(s)\n\n", len(resp.Files))
	for _, f := range resp.Files {
		fmt.Fprintf(&b, "- %s\n  ID: %s | Modified: %s\n  Link: %s\n",
			f.Name, f.Id, f.ModifiedTime, f.WebViewLink)
	}
	if resp.NextPageToken != "" {
		fmt.Fprintf(&b, "\nNext page token: %s\n", resp.NextPageToken)
	}
	return b.String(), nil
}

func registerList(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("list_spreadsheets",
		mcp.WithDescription("Lists Google Sheets in the user's Drive (filters by Sheet MIME type). Optional extra Drive query is AND-combined."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("query", mcp.Description("Optional extra Drive query terms")),
		mcp.WithNumber("page_size", mcp.Description("Max results, default 10, max 100")),
		mcp.WithString("page_token", mcp.Description("Pagination token for next page")),
	)
	scopes := []string{auth.DriveReadonlyScope}
	reg.Record("list_spreadsheets", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("list_spreadsheets", "Sheets", scopes, driveFactory, c, email, ListSpreadsheets))
}
