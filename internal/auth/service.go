// Package auth provides OAuth 2.0 flow, credential storage, and scope helpers
// for Google Workspace MCP tools.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"google.golang.org/api/option"

	"github.com/sausheong/go_gws_mcp/internal/core/apierror"
	"github.com/sausheong/go_gws_mcp/internal/core/mcpcontext"
)

// ServiceFactory builds a Google API client (Gmail, Drive, Docs, ...) from
// a context and option.ClientOptions (typically a TokenSource).
type ServiceFactory[S any] func(ctx context.Context, opts ...option.ClientOption) (S, error)

// ToolHandler is the body shape every Google-service tool implementation has.
// `svc` and `userEmail` are injected; the body is pure logic.
type ToolHandler[S any, T any] func(
	ctx context.Context,
	svc S,
	userEmail string,
	args T,
) (string, error)

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

// RequireGoogleService produces an mcp.ToolHandlerFunc that:
//  1. Binds args into T
//  2. Resolves user email (request -> ctx -> client default)
//  3. Loads + refreshes credentials via OAuthClient.GetCredentials
//  4. Builds the Google service via the supplied factory
//  5. Calls handler(ctx, svc, userEmail, args)
//  6. Wraps Google API errors via apierror.Format
//
// On AuthRequiredError, returns the LLM-targeted ACTION REQUIRED prose with
// auth URL via StartAuthFlow. On other errors, returns an error result.
func RequireGoogleService[S any, T any](
	toolName string,
	serviceName string,
	requiredScopes []string,
	factory ServiceFactory[S],
	client *OAuthClient,
	defaultEmail string,
	handler ToolHandler[S, T],
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
				flow, ferr := client.StartAuthFlow(userEmail, serviceName)
				if ferr != nil {
					slog.Warn("StartAuthFlow failed", "tool", toolName, "user", userEmail, "err", ferr)
					return mcp.NewToolResultText(authErr.Message), nil
				}
				return mcp.NewToolResultText(flow.Message), nil
			}
			return mcp.NewToolResultError(fmt.Sprintf("auth error: %v", err)), nil
		}

		ts := client.Config.TokenSource(ctx, token)
		svc, err := factory(ctx, option.WithTokenSource(ts))
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
