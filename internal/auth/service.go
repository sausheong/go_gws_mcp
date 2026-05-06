package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gmailapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/sausheong/go_gws_mcp/internal/core/apierror"
	"github.com/sausheong/go_gws_mcp/internal/core/mcpcontext"
)

// GmailHandler is the body shape every Gmail tool implementation has.
// `svc` and `userEmail` are injected; the body is pure logic.
type GmailHandler[T any] func(
	ctx context.Context,
	svc *gmailapi.Service,
	userEmail string,
	args T,
) (string, error)

// extractUserEmail pulls user_google_email from request arguments. We use it
// because mcp-go arguments come in as map[string]any and we need both the
// typed args struct (for the handler) and the email (before binding).
func extractUserEmail(req mcp.CallToolRequest, defaultEmail string) string {
	if req.Params.Arguments != nil {
		if m, ok := req.Params.Arguments.(map[string]any); ok {
			if v, ok := m["user_google_email"].(string); ok && v != "" {
				return v
			}
		}
	}
	return defaultEmail
}

// bindArgs deserializes req.Params.Arguments into T using a JSON round-trip.
// mcp-go's arguments are map[string]any; this is the simplest reliable path.
func bindArgs[T any](req mcp.CallToolRequest) (T, error) {
	var out T
	raw := req.Params.Arguments
	if raw == nil {
		return out, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, err
	}
	return out, nil
}

// RequireGmailService produces an mcp.ToolHandlerFunc that:
//  1. Binds args into T
//  2. Resolves user email (request -> ctx -> client default)
//  3. Loads + refreshes credentials via OAuthClient.GetCredentials
//  4. Builds gmail.Service
//  5. Calls handler(ctx, svc, userEmail, args)
//  6. Wraps Google API errors via apierror.Format
//
// On AuthRequiredError, returns the formatted instructions as a tool result.
// On other errors, returns an error result.
func RequireGmailService[T any](
	toolName string,
	requiredScopes []string,
	client *OAuthClient,
	defaultEmail string,
	handler GmailHandler[T],
) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := bindArgs[T](req)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		userEmail := extractUserEmail(req, defaultEmail)
		if userEmail == "" {
			if v, ok := mcpcontext.UserEmail(ctx); ok {
				userEmail = v
			}
		}
		if userEmail == "" {
			return mcp.NewToolResultError("user_google_email is required"), nil
		}
		ctx = mcpcontext.WithUserEmail(ctx, userEmail)

		token, err := client.GetCredentials(ctx, userEmail, requiredScopes)
		if err != nil {
			var authErr *AuthRequiredError
			if errors.As(err, &authErr) {
				return mcp.NewToolResultText(authErr.Message), nil
			}
			return mcp.NewToolResultError(fmt.Sprintf("auth error: %v", err)), nil
		}

		ts := client.Config.TokenSource(ctx, token)
		svc, err := gmailapi.NewService(ctx, option.WithTokenSource(ts))
		if err != nil {
			return mcp.NewToolResultError(apierror.Format(toolName, err)), nil
		}

		result, err := handler(ctx, svc, userEmail, args)
		if err != nil {
			slog.Warn("tool handler error", "tool", toolName, "user", userEmail, "err", err)
			return mcp.NewToolResultText(apierror.Format(toolName, err)), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}
