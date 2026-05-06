package drive

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

// GetContentArgs is the arg shape for get_drive_file_content.
type GetContentArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
	FileID          string `json:"file_id"`
}

// GetDriveFileContent inlines text/native content with metadata header.
// Native Google files (Doc/Sheet/Slides) are exported; text-like files are
// downloaded raw; binaries return a hint pointing to the web view link.
func GetDriveFileContent(ctx context.Context, svc *driveapi.Service, userEmail string, a GetContentArgs) (string, error) {
	if a.FileID == "" {
		return "", errors.New("file_id is required")
	}

	meta, err := svc.Files.Get(a.FileID).
		Fields("id, name, mimeType, size, webViewLink").
		Context(ctx).Do()
	if err != nil {
		return "", err
	}
	if meta.MimeType == MimeTypeFolder {
		return "", fmt.Errorf("file %s is a folder; use list_drive_items instead", a.FileID)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "File: %s\nID: %s\nMIME: %s\n", meta.Name, meta.Id, meta.MimeType)
	if meta.WebViewLink != "" {
		fmt.Fprintf(&b, "Link: %s\n", meta.WebViewLink)
	}
	fmt.Fprintln(&b, "---")

	if export := exportMimeFor(meta.MimeType); export != "" {
		resp, err := svc.Files.Export(a.FileID, export).Context(ctx).Download()
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, MaxContentBytes))
		if err != nil {
			return "", err
		}
		b.Write(body)
		return b.String(), nil
	}

	if isTextLike(meta.MimeType) {
		resp, err := svc.Files.Get(a.FileID).Context(ctx).Download()
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, MaxContentBytes))
		if err != nil {
			return "", err
		}
		b.Write(body)
		return b.String(), nil
	}

	fmt.Fprintf(&b, "Binary file (%s) — content not inlined. Use the link above to download.", meta.MimeType)
	return b.String(), nil
}

func registerGetContent(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("get_drive_file_content",
		mcp.WithDescription("Retrieves a Drive file's content. Google Docs/Sheets/Slides are exported (text/CSV/text); text-like files are downloaded raw; binaries return a hint with the web view link."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("file_id", mcp.Required(), mcp.Description("Drive file ID")),
	)
	scopes := []string{auth.DriveReadonlyScope}
	reg.Record("get_drive_file_content", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("get_drive_file_content", "Drive", scopes, driveFactory, c, email, GetDriveFileContent))
}
