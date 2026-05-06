package auth

import (
	"sort"
	"testing"
)

func TestHasRequiredScopes_DirectMatch(t *testing.T) {
	if !HasRequiredScopes([]string{GmailReadonlyScope}, []string{GmailReadonlyScope}) {
		t.Fatal("direct match should succeed")
	}
}

func TestHasRequiredScopes_MultipleRequiredAllPresent(t *testing.T) {
	avail := []string{GmailReadonlyScope, GmailSendScope}
	req := []string{GmailReadonlyScope, GmailSendScope}
	if !HasRequiredScopes(avail, req) {
		t.Fatal("all required scopes present should succeed")
	}
}

func TestHasRequiredScopes_MissingScopeFails(t *testing.T) {
	if HasRequiredScopes([]string{GmailReadonlyScope}, []string{GmailSendScope}) {
		t.Fatal("missing scope should fail")
	}
}

func TestHasRequiredScopes_ModifyCoversReadonly(t *testing.T) {
	if !HasRequiredScopes([]string{GmailModifyScope}, []string{GmailReadonlyScope}) {
		t.Fatal("gmail.modify should cover gmail.readonly via hierarchy")
	}
}

func TestHasRequiredScopes_ModifyCoversSendComposeLabels(t *testing.T) {
	avail := []string{GmailModifyScope}
	req := []string{GmailSendScope, GmailComposeScope, GmailLabelsScope}
	if !HasRequiredScopes(avail, req) {
		t.Fatal("gmail.modify should cover send/compose/labels")
	}
}

func TestHasRequiredScopes_NarrowDoesNotCoverBroad(t *testing.T) {
	if HasRequiredScopes([]string{GmailReadonlyScope}, []string{GmailModifyScope}) {
		t.Fatal("readonly should not cover modify")
	}
}

func TestScopesForTools_IncludesBaseAndPerService(t *testing.T) {
	got := ScopesForTools([]string{"gmail"})
	sort.Strings(got)
	want := []string{
		GmailComposeScope, GmailLabelsScope, GmailModifyScope,
		GmailReadonlyScope, GmailSendScope, GmailSettingsScope,
		OpenIDScope, UserinfoEmailScope, UserinfoProfileScope,
	}
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("got %d scopes, want %d: got=%v", len(got), len(want), got)
	}
	for i, s := range want {
		if got[i] != s {
			t.Fatalf("scope mismatch at %d: got %s, want %s", i, got[i], s)
		}
	}
}

func TestHasRequiredScopes_DriveCoversReadonly(t *testing.T) {
	if !HasRequiredScopes([]string{DriveScope}, []string{DriveReadonlyScope}) {
		t.Fatal("drive should cover drive.readonly via hierarchy")
	}
}

func TestHasRequiredScopes_DriveCoversFile(t *testing.T) {
	if !HasRequiredScopes([]string{DriveScope}, []string{DriveFileScope}) {
		t.Fatal("drive should cover drive.file via hierarchy")
	}
}

func TestScopesForTools_Drive(t *testing.T) {
	got := ScopesForTools([]string{"drive"})
	want := map[string]bool{
		DriveReadonlyScope:   true,
		DriveFileScope:       true,
		DriveScope:           true,
		OpenIDScope:          true,
		UserinfoEmailScope:   true,
		UserinfoProfileScope: true,
	}
	if len(got) != len(want) {
		t.Fatalf("got %d scopes, want %d: got=%v", len(got), len(want), got)
	}
	for _, s := range got {
		if !want[s] {
			t.Fatalf("unexpected scope: %s (got=%v)", s, got)
		}
	}
}
