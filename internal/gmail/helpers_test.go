package gmail

import (
	"encoding/base64"
	"strings"
	"testing"

	gmailapi "google.golang.org/api/gmail/v1"
)

func TestExtractTextBody_PlainOnly(t *testing.T) {
	body := "Hello world"
	encoded := base64.URLEncoding.EncodeToString([]byte(body))
	payload := &gmailapi.MessagePart{
		MimeType: "text/plain",
		Body:     &gmailapi.MessagePartBody{Data: encoded},
	}
	got := ExtractTextBody(payload)
	if got != "Hello world" {
		t.Fatalf("got %q, want %q", got, body)
	}
}

func TestExtractTextBody_PrefersPlainOverHTML(t *testing.T) {
	plain := base64.URLEncoding.EncodeToString([]byte("plain body"))
	html := base64.URLEncoding.EncodeToString([]byte("<p>html</p>"))
	payload := &gmailapi.MessagePart{
		MimeType: "multipart/alternative",
		Parts: []*gmailapi.MessagePart{
			{MimeType: "text/html", Body: &gmailapi.MessagePartBody{Data: html}},
			{MimeType: "text/plain", Body: &gmailapi.MessagePartBody{Data: plain}},
		},
	}
	got := ExtractTextBody(payload)
	if got != "plain body" {
		t.Fatalf("got %q, want plain body", got)
	}
}

func TestExtractTextBody_FallsBackToHTML(t *testing.T) {
	html := base64.URLEncoding.EncodeToString([]byte("<p>only html</p>"))
	payload := &gmailapi.MessagePart{
		MimeType: "multipart/alternative",
		Parts: []*gmailapi.MessagePart{
			{MimeType: "text/html", Body: &gmailapi.MessagePartBody{Data: html}},
		},
	}
	got := ExtractTextBody(payload)
	if !strings.Contains(got, "only html") {
		t.Fatalf("got %q", got)
	}
}

func TestHeaderValue(t *testing.T) {
	headers := []*gmailapi.MessagePartHeader{
		{Name: "From", Value: "alice@example.com"},
		{Name: "Subject", Value: "Hi"},
	}
	if v := HeaderValue(headers, "From"); v != "alice@example.com" {
		t.Errorf("From = %q", v)
	}
	if v := HeaderValue(headers, "from"); v != "alice@example.com" {
		t.Errorf("case-insensitive lookup failed: %q", v)
	}
	if v := HeaderValue(headers, "Missing"); v != "" {
		t.Errorf("missing header should return empty: %q", v)
	}
}
