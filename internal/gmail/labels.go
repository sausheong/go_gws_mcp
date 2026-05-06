package gmail

import (
	"context"
	"fmt"
	"strings"

	gmailapi "google.golang.org/api/gmail/v1"
)

// LabelsArgs is the arg shape for list_gmail_labels (only user email).
type LabelsArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
}

// ListGmailLabels returns the user's Gmail labels.
func ListGmailLabels(ctx context.Context, svc *gmailapi.Service, userEmail string, a LabelsArgs) (string, error) {
	resp, err := svc.Users.Labels.List("me").Context(ctx).Do()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Labels for %s (%d total):\n\n", userEmail, len(resp.Labels))
	for _, l := range resp.Labels {
		fmt.Fprintf(&b, "- %s [%s] (id: %s)\n", l.Name, l.Type, l.Id)
	}
	return b.String(), nil
}
