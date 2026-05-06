package slides

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	slidesapi "google.golang.org/api/slides/v1"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

// GetPageArgs is the arg shape for get_page.
type GetPageArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
	PresentationID  string `json:"presentation_id"`
	PageObjectID    string `json:"page_object_id"`
}

// GetPage returns the page structure summary for a single slide.
func GetPage(ctx context.Context, svc *slidesapi.Service, userEmail string, a GetPageArgs) (string, error) {
	if a.PresentationID == "" {
		return "", errors.New("presentation_id is required")
	}
	if a.PageObjectID == "" {
		return "", errors.New("page_object_id is required")
	}

	page, err := svc.Presentations.Pages.Get(a.PresentationID, a.PageObjectID).Context(ctx).Do()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Page: %s\nType: %s\nElements: %d\n---\n",
		page.ObjectId, page.PageType, len(page.PageElements))
	for _, el := range page.PageElements {
		kind := "unknown"
		switch {
		case el.Shape != nil:
			kind = "Shape"
		case el.Image != nil:
			kind = "Image"
		case el.Video != nil:
			kind = "Video"
		case el.Line != nil:
			kind = "Line"
		case el.Table != nil:
			kind = "Table"
		case el.WordArt != nil:
			kind = "WordArt"
		case el.SheetsChart != nil:
			kind = "SheetsChart"
		case el.ElementGroup != nil:
			kind = "Group"
		}
		fmt.Fprintf(&b, "- %s [%s] %q\n", el.ObjectId, kind, el.Title)
	}
	return b.String(), nil
}

func registerGetPage(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("get_page",
		mcp.WithDescription("Returns a single slide's structure summary (element IDs, types, titles)."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("presentation_id", mcp.Required(), mcp.Description("Presentation ID")),
		mcp.WithString("page_object_id", mcp.Required(), mcp.Description("Slide page object ID (from get_presentation)")),
	)
	scopes := []string{auth.SlidesReadonlyScope}
	reg.Record("get_page", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("get_page", "Slides", scopes, slidesFactory, c, email, GetPage))
}
