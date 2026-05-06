package gmail

import (
	"context"
	"fmt"
	"strings"

	gmailapi "google.golang.org/api/gmail/v1"
)

// GetMessageArgs is the arg shape for get_gmail_message_content.
type GetMessageArgs struct {
	MessageID       string `json:"message_id"`
	UserGoogleEmail string `json:"user_google_email"`
}

// GetGmailMessageContent fetches a single message and returns subject/from/body.
func GetGmailMessageContent(ctx context.Context, svc *gmailapi.Service, userEmail string, a GetMessageArgs) (string, error) {
	if a.MessageID == "" {
		return "", fmt.Errorf("message_id is required")
	}
	msg, err := svc.Users.Messages.Get("me", a.MessageID).Format("full").Context(ctx).Do()
	if err != nil {
		return "", err
	}

	subject := HeaderValue(msg.Payload.Headers, "Subject")
	from := HeaderValue(msg.Payload.Headers, "From")
	to := HeaderValue(msg.Payload.Headers, "To")
	date := HeaderValue(msg.Payload.Headers, "Date")
	body := ExtractTextBody(msg.Payload)

	var b strings.Builder
	fmt.Fprintf(&b, "Message ID: %s\nThread ID: %s\nUser: %s\n\n", msg.Id, msg.ThreadId, userEmail)
	fmt.Fprintf(&b, "Subject: %s\nFrom: %s\nTo: %s\nDate: %s\n\n", subject, from, to, date)
	fmt.Fprintf(&b, "Body:\n%s\n", body)
	return b.String(), nil
}
