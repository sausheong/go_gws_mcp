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

// GetArgs is the arg shape for get_presentation.
type GetArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
	PresentationID  string `json:"presentation_id"`
}

// GetPresentation returns the presentation title + slide inventory (page IDs +
// element counts). Does not return full per-element structure — use get_page for that.
func GetPresentation(ctx context.Context, svc *slidesapi.Service, userEmail string, a GetArgs) (string, error) {
	if a.PresentationID == "" {
		return "", errors.New("presentation_id is required")
	}
	pres, err := svc.Presentations.Get(a.PresentationID).Context(ctx).Do()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Presentation: %s\nID: %s\nLocale: %s\nSlides: %d\n---\n",
		pres.Title, pres.PresentationId, pres.Locale, len(pres.Slides))
	for i, sl := range pres.Slides {
		fmt.Fprintf(&b, "- [%d] %s (%d element(s))\n",
			i+1, sl.ObjectId, len(sl.PageElements))
	}
	return b.String(), nil
}

func registerGet(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("get_presentation",
		mcp.WithDescription("Returns presentation title + slide inventory (page IDs and element counts). Use get_page for full per-slide structure."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("presentation_id", mcp.Required(), mcp.Description("Google Slides presentation ID")),
	)
	scopes := []string{auth.SlidesReadonlyScope}
	reg.Record("get_presentation", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("get_presentation", "Slides", scopes, slidesFactory, c, email, GetPresentation))
}
