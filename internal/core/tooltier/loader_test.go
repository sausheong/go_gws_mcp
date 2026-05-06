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

func TestResolveToolsFromTier_DriveCore(t *testing.T) {
	loader, err := New()
	if err != nil {
		t.Fatal(err)
	}
	tools, services, err := loader.ResolveToolsFromTier(TierCore, []string{"drive"})
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 1 || services[0] != "drive" {
		t.Fatalf("services: got %v, want [drive]", services)
	}
	want := map[string]bool{
		"search_drive_files":     true,
		"list_drive_items":       true,
		"get_drive_file_content": true,
		"create_drive_folder":    true,
		"create_drive_file":      true,
	}
	if len(tools) != len(want) {
		t.Fatalf("tools count: got %d, want %d (got=%v)", len(tools), len(want), tools)
	}
	for _, n := range tools {
		if !want[n] {
			t.Fatalf("unexpected tool: %s", n)
		}
	}
}

func TestResolveToolsFromTier_DocsCore(t *testing.T) {
	loader, err := New()
	if err != nil {
		t.Fatal(err)
	}
	tools, services, err := loader.ResolveToolsFromTier(TierCore, []string{"docs"})
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 1 || services[0] != "docs" {
		t.Fatalf("services: got %v, want [docs]", services)
	}
	want := map[string]bool{
		"search_docs":         true,
		"get_doc_content":     true,
		"get_doc_as_markdown": true,
		"create_doc":          true,
		"modify_doc_text":     true,
	}
	if len(tools) != len(want) {
		t.Fatalf("tools count: got %d, want %d (got=%v)", len(tools), len(want), tools)
	}
	for _, n := range tools {
		if !want[n] {
			t.Fatalf("unexpected tool: %s", n)
		}
	}
}
