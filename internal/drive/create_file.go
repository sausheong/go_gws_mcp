package drive

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	driveapi "google.golang.org/api/drive/v3"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

// CreateFileArgs is the arg shape for create_drive_file.
type CreateFileArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
	Name            string `json:"name"`
	MimeType        string `json:"mime_type,omitempty"` // default text/plain
	ParentID        string `json:"parent_id,omitempty"`
	Content         string `json:"content,omitempty"`
}

// CreateDriveFile creates a new file with optional inline text content.
func CreateDriveFile(ctx context.Context, svc *driveapi.Service, userEmail string, a CreateFileArgs) (string, error) {
	if a.Name == "" {
		return "", errors.New("name is required")
	}
	mime := a.MimeType
	if mime == "" {
		mime = "text/plain"
	}
	f := &driveapi.File{
		Name:     a.Name,
		MimeType: mime,
	}
	if a.ParentID != "" {
		f.Parents = []string{a.ParentID}
	}
	call := svc.Files.Create(f).Fields("id, name, mimeType, webViewLink")
	if a.Content != "" {
		call = call.Media(strings.NewReader(a.Content))
	}
	created, err := call.Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Created file %q (id: %s, mime: %s) — %s",
		created.Name, created.Id, created.MimeType, created.WebViewLink), nil
}

func registerCreateFile(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("create_drive_file",
		mcp.WithDescription("Creates a new Drive file with optional inline text content. Default MIME is text/plain."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("name", mcp.Required(), mcp.Description("File name")),
		mcp.WithString("mime_type", mcp.Description("MIME type (default text/plain)")),
		mcp.WithString("parent_id", mcp.Description("Parent folder ID (omit for My Drive root)")),
		mcp.WithString("content", mcp.Description("Initial text content (UTF-8 only)")),
	)
	scopes := []string{auth.DriveFileScope}
	reg.Record("create_drive_file", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("create_drive_file", "Drive", scopes, driveFactory, c, email, CreateDriveFile))
}
