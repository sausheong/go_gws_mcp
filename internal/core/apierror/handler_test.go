package apierror

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/googleapi"
)

func TestFormat_AccessNotConfigured(t *testing.T) {
	err := &googleapi.Error{
		Code:    403,
		Message: "Gmail API has not been used in project 12345 before...",
		Errors: []googleapi.ErrorItem{
			{Reason: "accessNotConfigured"},
		},
	}
	out := Format("search_gmail_messages", err)
	if !strings.Contains(out, "API is not enabled") {
		t.Errorf("expected enable-API hint, got: %s", out)
	}
	if !strings.Contains(out, "IMPORTANT - LLM:") {
		t.Errorf("expected LLM directive, got: %s", out)
	}
}

func TestFormat_AuthError401(t *testing.T) {
	err := &googleapi.Error{Code: 401, Message: "Invalid Credentials"}
	out := Format("get_gmail_message_content", err)
	if !strings.Contains(out, "re-authenticate") {
		t.Errorf("expected re-auth hint, got: %s", out)
	}
}

func TestFormat_GenericGoogleError(t *testing.T) {
	err := &googleapi.Error{Code: 400, Message: "Bad Request"}
	out := Format("send_gmail_message", err)
	if !strings.Contains(out, "API error in send_gmail_message") {
		t.Errorf("expected generic error format, got: %s", out)
	}
}

func TestFormat_NonGoogleError(t *testing.T) {
	out := Format("search_gmail_messages", errors.New("network down"))
	if !strings.Contains(out, "Unexpected error") {
		t.Errorf("expected unexpected wrapper, got: %s", out)
	}
}
