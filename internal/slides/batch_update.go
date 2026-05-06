package slides

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	slidesapi "google.golang.org/api/slides/v1"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

// BatchUpdateArgs is the arg shape for batch_update_presentation.
type BatchUpdateArgs struct {
	UserGoogleEmail string          `json:"user_google_email"`
	PresentationID  string          `json:"presentation_id"`
	Requests        json.RawMessage `json:"requests"` // JSON array of Slides API Request objects
}

// BatchUpdatePresentation applies a batch of Slides API requests to a presentation.
// `requests` is a JSON array of Request objects per the Slides API spec
// (see https://developers.google.com/slides/api/reference/rest/v1/presentations/request).
func BatchUpdatePresentation(ctx context.Context, svc *slidesapi.Service, userEmail string, a BatchUpdateArgs) (string, error) {
	if a.PresentationID == "" {
		return "", errors.New("presentation_id is required")
	}
	if len(a.Requests) == 0 {
		return "", errors.New("requests is required (JSON array of Slides API Request objects)")
	}

	var reqs []*slidesapi.Request
	if err := json.Unmarshal(a.Requests, &reqs); err != nil {
		return "", fmt.Errorf("requests must be a JSON array of Slides API Request objects: %w", err)
	}

	resp, err := svc.Presentations.BatchUpdate(a.PresentationID, &slidesapi.BatchUpdatePresentationRequest{
		Requests: reqs,
	}).Context(ctx).Do()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Applied %d request(s) to presentation %s (replies: %d)",
		len(reqs), resp.PresentationId, len(resp.Replies)), nil
}

func registerBatchUpdate(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("batch_update_presentation",
		mcp.WithDescription("Applies a batch of Slides API requests to a presentation. `requests` is a JSON array of Request objects (see https://developers.google.com/slides/api/reference/rest/v1/presentations/request)."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("presentation_id", mcp.Required(), mcp.Description("Presentation ID")),
		mcp.WithArray("requests",
			mcp.Required(),
			mcp.Description("Array of Slides API Request objects"),
			mcp.Items(map[string]any{"type": "object"}),
		),
	)
	scopes := []string{auth.SlidesScope}
	reg.Record("batch_update_presentation", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("batch_update_presentation", "Slides", scopes, slidesFactory, c, email, BatchUpdatePresentation))
}
