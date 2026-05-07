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

// ListArgs is the arg shape for list_drive_items.
type ListArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
	FolderID        string `json:"folder_id,omitempty"` // empty = root
	PageSize        int    `json:"page_size,omitempty"`
	PageToken       string `json:"page_token,omitempty"`
}

// ListDriveItems lists immediate children of a folder (or root).
func ListDriveItems(ctx context.Context, svc *driveapi.Service, userEmail string, a ListArgs) (string, error) {
	pageSize := a.PageSize
	if pageSize <= 0 {
		pageSize = 25
	}
	if pageSize > 100 {
		pageSize = 100
	}
	parent := a.FolderID
	if parent == "" {
		parent = "root"
	}
	q := fmt.Sprintf("'%s' in parents and trashed = false", escapeDriveString(parent))

	call := svc.Files.List().
		Q(q).
		PageSize(int64(pageSize)).
		Fields("nextPageToken, files(id, name, mimeType, modifiedTime)").
		Spaces("drive")
	if a.PageToken != "" {
		call = call.PageToken(a.PageToken)
	}
	resp, err := call.Context(ctx).Do()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Items in folder %q (user: %s)\n", parent, userEmail)
	fmt.Fprintf(&b, "Found %d item(s)\n\n", len(resp.Files))
	for _, f := range resp.Files {
		fmt.Fprintf(&b, "- %s [%s]\n  ID: %s | Modified: %s\n", f.Name, f.MimeType, f.Id, f.ModifiedTime)
	}
	if resp.NextPageToken != "" {
		fmt.Fprintf(&b, "\nNext page token: %s\n", resp.NextPageToken)
	}
	return b.String(), nil
}

func registerList(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("list_drive_items",
		mcp.WithDescription("Lists immediate children of a Drive folder. Pass folder_id='' (or omit) for the user's My Drive root."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("folder_id", mcp.Description("Folder ID; empty for root")),
		mcp.WithNumber("page_size", mcp.Description("Max results, default 25, max 100")),
		mcp.WithString("page_token", mcp.Description("Pagination token for next page")),
	)
	scopes := []string{auth.DriveReadonlyScope}
	reg.Record("list_drive_items", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("list_drive_items", "Drive", scopes, driveFactory, c, email, ListDriveItems))
}
