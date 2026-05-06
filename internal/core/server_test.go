package core

import (
	"reflect"
	"sort"
	"testing"
)

func TestRegistry_RecordsTools(t *testing.T) {
	r := NewRegistry()
	r.Record("search_gmail_messages", []string{"https://www.googleapis.com/auth/gmail.readonly"})
	r.Record("send_gmail_message", []string{"https://www.googleapis.com/auth/gmail.send"})

	names := r.Names()
	sort.Strings(names)
	want := []string{"search_gmail_messages", "send_gmail_message"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("got %v, want %v", names, want)
	}
}

func TestRegistry_FilterByEnabled_KeepsEnabledOnly(t *testing.T) {
	r := NewRegistry()
	r.Record("a", nil)
	r.Record("b", nil)
	r.Record("c", nil)

	enabled := map[string]struct{}{"a": {}, "c": {}}
	removed := r.computeRemovals(enabled, nil)
	sort.Strings(removed)
	if !reflect.DeepEqual(removed, []string{"b"}) {
		t.Fatalf("got removals %v, want [b]", removed)
	}
}

func TestRegistry_FilterByEnabled_NilMeansKeepAll(t *testing.T) {
	r := NewRegistry()
	r.Record("a", nil)
	r.Record("b", nil)
	removed := r.computeRemovals(nil, nil)
	if len(removed) != 0 {
		t.Fatalf("expected no removals, got %v", removed)
	}
}
