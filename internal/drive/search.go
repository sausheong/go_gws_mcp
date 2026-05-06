package drive

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

// SearchArgs is the arg shape for search_drive_files.
type SearchArgs struct {
	Query           string `json:"query"`
	UserGoogleEmail string `json:"user_google_email"`
	PageSize        int    `json:"page_size,omitempty"`
	PageToken       string `json:"page_token,omitempty"`
}

// SearchDriveFiles searches files using Drive query syntax.
func SearchDriveFiles(ctx context.Context, svc *driveapi.Service, userEmail string, a SearchArgs) (string, error) {
	pageSize := a.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	call := svc.Files.List().
		Q(a.Query).
		PageSize(int64(pageSize)).
		Fields("nextPageToken, files(id, name, mimeType, modifiedTime, size, webViewLink)").
		Spaces("drive")
	if a.PageToken != "" {
		call = call.PageToken(a.PageToken)
	}
	resp, err := call.Context(ctx).Do()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Drive search results for query: %q (user: %s)\n", a.Query, userEmail)
	fmt.Fprintf(&b, "Found %d file(s)\n\n", len(resp.Files))
	for _, f := range resp.Files {
		fmt.Fprintf(&b, "- %s\n", f.Name)
		fmt.Fprintf(&b, "  ID: %s | MIME: %s | Modified: %s\n", f.Id, f.MimeType, f.ModifiedTime)
		if f.WebViewLink != "" {
			fmt.Fprintf(&b, "  Link: %s\n", f.WebViewLink)
		}
	}
	if resp.NextPageToken != "" {
		fmt.Fprintf(&b, "\nNext page token: %s\n", resp.NextPageToken)
	}
	return b.String(), nil
}

func registerSearch(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("search_drive_files",
		mcp.WithDescription("Searches files in Google Drive using Drive query syntax (e.g., \"name contains 'budget'\", \"mimeType = 'application/pdf'\")."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Drive query string")),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithNumber("page_size", mcp.Description("Max results, default 10, max 100")),
		mcp.WithString("page_token", mcp.Description("Pagination token for next page")),
	)
	scopes := []string{auth.DriveReadonlyScope}
	reg.Record("search_drive_files", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("search_drive_files", "Drive", scopes, driveFactory, c, email, SearchDriveFiles))
}
