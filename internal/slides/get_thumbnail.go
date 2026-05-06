package slides

import (
	"context"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	slidesapi "google.golang.org/api/slides/v1"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

// GetThumbnailArgs is the arg shape for get_page_thumbnail.
type GetThumbnailArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
	PresentationID  string `json:"presentation_id"`
	PageObjectID    string `json:"page_object_id"`
	ThumbnailSize   string `json:"thumbnail_size,omitempty"` // "SMALL", "MEDIUM" (default), "LARGE"
}

// GetPageThumbnail returns the URL of a generated thumbnail for the given slide.
// Thumbnails are short-lived (~30 minutes per Google docs).
func GetPageThumbnail(ctx context.Context, svc *slidesapi.Service, userEmail string, a GetThumbnailArgs) (string, error) {
	if a.PresentationID == "" {
		return "", errors.New("presentation_id is required")
	}
	if a.PageObjectID == "" {
		return "", errors.New("page_object_id is required")
	}

	size := a.ThumbnailSize
	if size == "" {
		size = "MEDIUM"
	}
	if size != "SMALL" && size != "MEDIUM" && size != "LARGE" {
		return "", fmt.Errorf("thumbnail_size must be SMALL, MEDIUM, or LARGE (got %q)", size)
	}

	call := svc.Presentations.Pages.GetThumbnail(a.PresentationID, a.PageObjectID).
		ThumbnailPropertiesThumbnailSize(size).
		Context(ctx)
	thumb, err := call.Do()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Thumbnail: %s\nDimensions: %dx%d (size: %s)\nNote: URL is short-lived (~30 minutes).",
		thumb.ContentUrl, thumb.Width, thumb.Height, size), nil
}

func registerGetThumbnail(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("get_page_thumbnail",
		mcp.WithDescription("Returns a short-lived thumbnail URL for a slide. thumbnail_size: SMALL | MEDIUM (default) | LARGE."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("presentation_id", mcp.Required(), mcp.Description("Presentation ID")),
		mcp.WithString("page_object_id", mcp.Required(), mcp.Description("Slide page object ID")),
		mcp.WithString("thumbnail_size", mcp.Description("SMALL | MEDIUM (default) | LARGE")),
	)
	scopes := []string{auth.SlidesReadonlyScope}
	reg.Record("get_page_thumbnail", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("get_page_thumbnail", "Slides", scopes, slidesFactory, c, email, GetPageThumbnail))
}
