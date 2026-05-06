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

// CreateArgs is the arg shape for create_presentation.
type CreateArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
	Title           string `json:"title"`
}

// CreatePresentation creates a new Google Slides presentation with the given title.
func CreatePresentation(ctx context.Context, svc *slidesapi.Service, userEmail string, a CreateArgs) (string, error) {
	if a.Title == "" {
		return "", errors.New("title is required")
	}
	created, err := svc.Presentations.Create(&slidesapi.Presentation{Title: a.Title}).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	link := fmt.Sprintf("https://docs.google.com/presentation/d/%s/edit", created.PresentationId)
	return fmt.Sprintf("Created presentation %q (id: %s) — %s",
		created.Title, created.PresentationId, link), nil
}

func registerCreate(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("create_presentation",
		mcp.WithDescription("Creates a new Google Slides presentation with the given title (and a default first slide)."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Presentation title")),
	)
	scopes := []string{auth.SlidesScope}
	reg.Record("create_presentation", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("create_presentation", "Slides", scopes, slidesFactory, c, email, CreatePresentation))
}
