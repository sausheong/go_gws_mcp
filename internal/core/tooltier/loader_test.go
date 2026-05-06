package tooltier

import (
	"sort"
	"testing"
)

func TestResolveToolsFromTier_GmailCore(t *testing.T) {
	loader, err := New()
	if err != nil {
		t.Fatalf("loader init: %v", err)
	}
	tools, services, err := loader.ResolveToolsFromTier("core", []string{"gmail"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	wantTools := []string{
		"get_gmail_message_content",
		"get_gmail_messages_content_batch",
		"list_gmail_labels",
		"search_gmail_messages",
		"send_gmail_message",
	}
	sort.Strings(tools)
	sort.Strings(wantTools)
	if len(tools) != len(wantTools) {
		t.Fatalf("got %d tools (%v), want %d (%v)", len(tools), tools, len(wantTools), wantTools)
	}
	for i, want := range wantTools {
		if tools[i] != want {
			t.Errorf("tool[%d] = %s, want %s", i, tools[i], want)
		}
	}
	if len(services) != 1 || services[0] != "gmail" {
		t.Fatalf("services = %v, want [gmail]", services)
	}
}

func TestResolveToolsFromTier_ExtendedIsCumulative(t *testing.T) {
	loader, _ := New()
	tools, _, _ := loader.ResolveToolsFromTier("extended", []string{"gmail"})
	hasCore := false
	for _, tool := range tools {
		if tool == "search_gmail_messages" {
			hasCore = true
		}
	}
	if !hasCore {
		t.Fatal("extended should include core tools (cumulative)")
	}
}

func TestResolveToolsFromTier_UnknownTier(t *testing.T) {
	loader, _ := New()
	_, _, err := loader.ResolveToolsFromTier("bogus", []string{"gmail"})
	if err == nil {
		t.Fatal("expected error for unknown tier")
	}
}
