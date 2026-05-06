package mcpcontext

import (
	"context"
	"testing"
)

func TestUserEmail_RoundTrip(t *testing.T) {
	ctx := WithUserEmail(context.Background(), "alice@example.com")
	got, ok := UserEmail(ctx)
	if !ok || got != "alice@example.com" {
		t.Fatalf("got (%q, %v)", got, ok)
	}
}

func TestUserEmail_AbsentReturnsFalse(t *testing.T) {
	if _, ok := UserEmail(context.Background()); ok {
		t.Fatal("expected ok=false on bare context")
	}
}

func TestSessionID_RoundTrip(t *testing.T) {
	ctx := WithSessionID(context.Background(), "sess-123")
	got, ok := SessionID(ctx)
	if !ok || got != "sess-123" {
		t.Fatalf("got (%q, %v)", got, ok)
	}
}
