package auth

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestParsePermissions_ValidSingle(t *testing.T) {
	got, err := ParsePermissions([]string{"gmail:organize"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]string{"gmail": "organize"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParsePermissions_ValidMultiple(t *testing.T) {
	got, err := ParsePermissions([]string{"gmail:send", "drive:readonly"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["gmail"] != "send" || got["drive"] != "readonly" {
		t.Fatalf("got %v", got)
	}
}

func TestParsePermissions_BadFormat(t *testing.T) {
	_, err := ParsePermissions([]string{"gmail-organize"})
	if err == nil || !strings.Contains(err.Error(), "service:level") {
		t.Fatalf("want format error, got %v", err)
	}
}

func TestParsePermissions_UnknownService(t *testing.T) {
	_, err := ParsePermissions([]string{"unknownsvc:readonly"})
	if err == nil || !strings.Contains(err.Error(), "Unknown service") {
		t.Fatalf("want unknown service error, got %v", err)
	}
}

func TestParsePermissions_UnknownLevel(t *testing.T) {
	_, err := ParsePermissions([]string{"gmail:bogus"})
	if err == nil || !strings.Contains(err.Error(), "Unknown level") {
		t.Fatalf("want unknown level error, got %v", err)
	}
}

func TestParsePermissions_DuplicateService(t *testing.T) {
	_, err := ParsePermissions([]string{"gmail:send", "gmail:readonly"})
	if err == nil || !strings.Contains(err.Error(), "Duplicate") {
		t.Fatalf("want duplicate error, got %v", err)
	}
}

func TestScopesForPermission_GmailOrganize(t *testing.T) {
	got, err := ScopesForPermission("gmail", "organize")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(got)
	want := []string{GmailLabelsScope, GmailModifyScope, GmailReadonlyScope}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestScopesForPermission_GmailSendIsCumulative(t *testing.T) {
	got, _ := ScopesForPermission("gmail", "send")
	gotSet := make(map[string]bool)
	for _, s := range got {
		gotSet[s] = true
	}
	for _, must := range []string{GmailReadonlyScope, GmailLabelsScope, GmailModifyScope, GmailComposeScope, GmailSendScope} {
		if !gotSet[must] {
			t.Errorf("gmail:send missing cumulative scope %s", must)
		}
	}
}
