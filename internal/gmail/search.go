package gmail

import (
	"context"
	"fmt"
	"strings"

	gmailapi "google.golang.org/api/gmail/v1"
)

// SearchArgs is the arg shape for search_gmail_messages.
type SearchArgs struct {
	Query           string `json:"query"`
	UserGoogleEmail string `json:"user_google_email"`
	PageSize        int    `json:"page_size,omitempty"`
	PageToken       string `json:"page_token,omitempty"`
}

// SearchGmailMessages lists message IDs matching a Gmail search query.
func SearchGmailMessages(ctx context.Context, svc *gmailapi.Service, userEmail string, a SearchArgs) (string, error) {
	pageSize := a.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	call := svc.Users.Messages.List("me").Q(a.Query).MaxResults(int64(pageSize))
	if a.PageToken != "" {
		call = call.PageToken(a.PageToken)
	}
	resp, err := call.Context(ctx).Do()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Search results for query: %q (user: %s)\n", a.Query, userEmail)
	fmt.Fprintf(&b, "Found %d message(s)\n\n", len(resp.Messages))
	for _, m := range resp.Messages {
		fmt.Fprintf(&b, "- Message ID: %s | Thread ID: %s\n", m.Id, m.ThreadId)
		fmt.Fprintf(&b, "  URL: https://mail.google.com/mail/u/0/#inbox/%s\n", m.Id)
	}
	if resp.NextPageToken != "" {
		fmt.Fprintf(&b, "\nNext page token: %s\n", resp.NextPageToken)
	}
	return b.String(), nil
}
