package gmail

import (
	"context"
	"fmt"
	"strings"
	"sync"

	gmailapi "google.golang.org/api/gmail/v1"
)

const (
	batchConcurrency = 5
	batchMaxIDs      = 100
)

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

// GetGmailMessagesContentBatch fetches up to batchMaxIDs messages in parallel.
func GetGmailMessagesContentBatch(ctx context.Context, svc *gmailapi.Service, userEmail string, a BatchGetArgs) (string, error) {
	if len(a.MessageIDs) == 0 {
		return "", fmt.Errorf("message_ids is required")
	}
	if len(a.MessageIDs) > batchMaxIDs {
		return "", fmt.Errorf("message_ids exceeds limit of %d (got %d)", batchMaxIDs, len(a.MessageIDs))
	}

	results := make([]batchResult, len(a.MessageIDs))
	sem := make(chan struct{}, batchConcurrency)
	var wg sync.WaitGroup

	for i, id := range a.MessageIDs {
		// Acquire BEFORE spawning so we cap the total live goroutine count
		// at batchConcurrency (not the total number of message IDs).
		sem <- struct{}{}
		i, id := i, id
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					results[i] = batchResult{idx: i, err: fmt.Errorf("panic fetching %s: %v", id, r)}
				}
			}()

			msg, err := svc.Users.Messages.Get("me", id).Format("metadata").Context(ctx).Do()
			if err != nil {
				results[i] = batchResult{idx: i, err: err}
				return
			}
			if msg.Payload == nil {
				results[i] = batchResult{idx: i, err: fmt.Errorf("message %s returned no payload", id)}
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
