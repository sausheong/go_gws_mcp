package gmail

import (
	"context"
	"fmt"
	"strings"
	"sync"

	gmailapi "google.golang.org/api/gmail/v1"
)

const batchConcurrency = 5

// BatchGetArgs is the arg shape for get_gmail_messages_content_batch.
type BatchGetArgs struct {
	MessageIDs      []string `json:"message_ids"`
	UserGoogleEmail string   `json:"user_google_email"`
}

type batchResult struct {
	idx int
	out string
	err error
}

// GetGmailMessagesContentBatch fetches up to N messages in parallel.
func GetGmailMessagesContentBatch(ctx context.Context, svc *gmailapi.Service, userEmail string, a BatchGetArgs) (string, error) {
	if len(a.MessageIDs) == 0 {
		return "", fmt.Errorf("message_ids is required")
	}

	results := make([]batchResult, len(a.MessageIDs))
	sem := make(chan struct{}, batchConcurrency)
	var wg sync.WaitGroup

	for i, id := range a.MessageIDs {
		i, id := i, id
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			msg, err := svc.Users.Messages.Get("me", id).Format("metadata").Context(ctx).Do()
			if err != nil {
				results[i] = batchResult{idx: i, err: err}
				return
			}
			subject := HeaderValue(msg.Payload.Headers, "Subject")
			from := HeaderValue(msg.Payload.Headers, "From")
			results[i] = batchResult{
				idx: i,
				out: fmt.Sprintf("- %s | From: %s | Subject: %s", msg.Id, from, subject),
			}
		}()
	}
	wg.Wait()

	var b strings.Builder
	fmt.Fprintf(&b, "Batch results (%d messages, user: %s):\n\n", len(a.MessageIDs), userEmail)
	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(&b, "- ERROR fetching %s: %v\n", a.MessageIDs[r.idx], r.err)
			continue
		}
		fmt.Fprintln(&b, r.out)
	}
	return b.String(), nil
}
