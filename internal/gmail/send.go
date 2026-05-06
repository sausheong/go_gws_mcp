package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	gmailapi "google.golang.org/api/gmail/v1"
)

// SendArgs is the arg shape for send_gmail_message.
type SendArgs struct {
	To              string `json:"to"`
	Subject         string `json:"subject"`
	Body            string `json:"body"`
	UserGoogleEmail string `json:"user_google_email"`
	Cc              string `json:"cc,omitempty"`
	Bcc             string `json:"bcc,omitempty"`
}

// SendGmailMessage composes a plain-text RFC 822 message and sends it.
func SendGmailMessage(ctx context.Context, svc *gmailapi.Service, userEmail string, a SendArgs) (string, error) {
	if a.To == "" || a.Subject == "" {
		return "", fmt.Errorf("to and subject are required")
	}
	raw := buildRFC822Message(userEmail, a)
	encoded := base64.URLEncoding.EncodeToString([]byte(raw))

	msg, err := svc.Users.Messages.Send("me", &gmailapi.Message{Raw: encoded}).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Sent message ID: %s\nThread ID: %s", msg.Id, msg.ThreadId), nil
}

func buildRFC822Message(from string, a SendArgs) string {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", a.To)
	if a.Cc != "" {
		fmt.Fprintf(&b, "Cc: %s\r\n", a.Cc)
	}
	if a.Bcc != "" {
		fmt.Fprintf(&b, "Bcc: %s\r\n", a.Bcc)
	}
	fmt.Fprintf(&b, "Subject: %s\r\n", a.Subject)
	fmt.Fprintf(&b, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	fmt.Fprintf(&b, "\r\n%s", a.Body)
	return b.String()
}
