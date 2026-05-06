package drive

import (
	"context"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	driveapi "google.golang.org/api/drive/v3"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

// CreateFolderArgs is the arg shape for create_drive_folder.
type CreateFolderArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
	Name            string `json:"name"`
	ParentID        string `json:"parent_id,omitempty"`
}

// CreateDriveFolder creates a new folder, optionally inside a parent.
func CreateDriveFolder(ctx context.Context, svc *driveapi.Service, userEmail string, a CreateFolderArgs) (string, error) {
	if a.Name == "" {
		return "", errors.New("name is required")
	}
	f := &driveapi.File{
		Name:     a.Name,
		MimeType: MimeTypeFolder,
	}
	if a.ParentID != "" {
		f.Parents = []string{a.ParentID}
	}
	created, err := svc.Files.Create(f).
		Fields("id, name, webViewLink").
		Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Created folder %q (id: %s) — %s", created.Name, created.Id, created.WebViewLink), nil
}

func registerCreateFolder(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("create_drive_folder",
		mcp.WithDescription("Creates a new Drive folder, optionally inside a parent folder."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Folder name")),
		mcp.WithString("parent_id", mcp.Description("Parent folder ID (omit for My Drive root)")),
	)
	scopes := []string{auth.DriveFileScope}
	reg.Record("create_drive_folder", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("create_drive_folder", "Drive", scopes, driveFactory, c, email, CreateDriveFolder))
}
