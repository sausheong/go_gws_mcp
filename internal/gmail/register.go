package gmail

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gmailapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

func gmailFactory(ctx context.Context, opts ...option.ClientOption) (*gmailapi.Service, error) {
	return gmailapi.NewService(ctx, opts...)
}

// RegisterTools wires all Gmail tools onto srv and records them in registry.
func RegisterTools(srv *server.MCPServer, registry *core.Registry, oauthClient *auth.OAuthClient, defaultEmail string) {
	registerSearch(srv, registry, oauthClient, defaultEmail)
	registerGet(srv, registry, oauthClient, defaultEmail)
	registerBatch(srv, registry, oauthClient, defaultEmail)
	registerSend(srv, registry, oauthClient, defaultEmail)
	registerLabels(srv, registry, oauthClient, defaultEmail)
}

func registerSearch(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("search_gmail_messages",
		mcp.WithDescription("Searches messages in a user's Gmail account based on a query. Supports standard Gmail search operators."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Gmail search query (e.g., 'is:unread')")),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithNumber("page_size", mcp.Description("Max results, default 10")),
		mcp.WithString("page_token", mcp.Description("Pagination token for next page")),
	)
	scopes := []string{auth.GmailReadonlyScope}
	reg.Record("search_gmail_messages", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("search_gmail_messages", "Gmail", scopes, gmailFactory, c, email, SearchGmailMessages))
}

func registerGet(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("get_gmail_message_content",
		mcp.WithDescription("Retrieves the full content (subject, from, to, date, body) of a specific Gmail message."),
		mcp.WithString("message_id", mcp.Required(), mcp.Description("The unique Gmail message ID")),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
	)
	scopes := []string{auth.GmailReadonlyScope}
	reg.Record("get_gmail_message_content", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("get_gmail_message_content", "Gmail", scopes, gmailFactory, c, email, GetGmailMessageContent))
}

func registerBatch(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("get_gmail_messages_content_batch",
		mcp.WithDescription("Fetches metadata (id, from, subject) for multiple message IDs in parallel."),
		mcp.WithArray("message_ids",
			mcp.Required(),
			mcp.Description("List of Gmail message IDs to fetch"),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
	)
	scopes := []string{auth.GmailReadonlyScope}
	reg.Record("get_gmail_messages_content_batch", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("get_gmail_messages_content_batch", "Gmail", scopes, gmailFactory, c, email, GetGmailMessagesContentBatch))
}

func registerSend(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("send_gmail_message",
		mcp.WithDescription("Sends a plain-text email from the authenticated Gmail account."),
		mcp.WithString("to", mcp.Required(), mcp.Description("Recipient email address(es), comma-separated")),
		mcp.WithString("subject", mcp.Required(), mcp.Description("Email subject")),
		mcp.WithString("body", mcp.Required(), mcp.Description("Plain-text message body")),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("Sender's Google email address")),
		mcp.WithString("cc", mcp.Description("CC recipients")),
		mcp.WithString("bcc", mcp.Description("BCC recipients")),
	)
	scopes := []string{auth.GmailSendScope}
	reg.Record("send_gmail_message", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("send_gmail_message", "Gmail", scopes, gmailFactory, c, email, SendGmailMessage))
}

func registerLabels(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("list_gmail_labels",
		mcp.WithDescription("Lists all labels (system and user-defined) in the user's Gmail account."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
	)
	scopes := []string{auth.GmailReadonlyScope}
	reg.Record("list_gmail_labels", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("list_gmail_labels", "Gmail", scopes, gmailFactory, c, email, ListGmailLabels))
}
