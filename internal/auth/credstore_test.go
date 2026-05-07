package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func newCred(access, refresh string, scopes ...string) *StoredCredential {
	return &StoredCredential{
		Token: &oauth2.Token{
			AccessToken:  access,
			RefreshToken: refresh,
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(time.Hour).Round(time.Second),
		},
		Scopes: scopes,
	}
}

func TestLocalDirectoryStore_StoreAndGet(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalDirectoryStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	cred := newCred("atk", "rtk",
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/drive.readonly",
	)
	if err := store.Store("alice@example.com", cred); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Token.AccessToken != "atk" || got.Token.RefreshToken != "rtk" {
		t.Fatalf("token mismatch: %+v", got.Token)
	}
	if len(got.Scopes) != 2 {
		t.Fatalf("want 2 scopes, got %v", got.Scopes)
	}
}

func TestLocalDirectoryStore_GetMissingReturnsNilNoError(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewLocalDirectoryStore(dir)
	cred, err := store.Get("nobody@example.com")
	if err != nil {
		t.Fatalf("want nil err, got %v", err)
	}
	if cred != nil {
		t.Fatal("want nil cred for missing user")
	}
}

func TestLocalDirectoryStore_FileMode0600(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewLocalDirectoryStore(dir)
	store.Store("bob@example.com", newCred("x", ""))

	matches, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	if len(matches) != 1 {
		t.Fatalf("want 1 file, got %v", matches)
	}
	info, _ := os.Stat(matches[0])
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("want 0600, got %o", info.Mode().Perm())
	}
}

func TestLocalDirectoryStore_Delete(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewLocalDirectoryStore(dir)
	store.Store("c@example.com", newCred("x", ""))
	if err := store.Delete("c@example.com"); err != nil {
		t.Fatal(err)
	}
	got, _ := store.Get("c@example.com")
	if got != nil {
		t.Fatal("expected gone after delete")
	}
}

func TestLocalDirectoryStore_List(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewLocalDirectoryStore(dir)
	store.Store("a@example.com", newCred("x", ""))
	store.Store("b@example.com", newCred("x", ""))
	users, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatalf("want 2 users, got %v", users)
	}
}

func TestLocalDirectoryStore_RejectsPathTraversalEmail(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewLocalDirectoryStore(dir)
	err := store.Store("../../../etc/passwd", newCred("x", ""))
	if err == nil {
		t.Fatal("want rejection for traversal-style email")
	}
}

// TestLocalDirectoryStore_BackwardCompat_BareToken verifies that an existing
// credential file written before the StoredCredential refactor (i.e. a bare
// oauth2.Token JSON object) is still readable. Scopes will be empty on read,
// which is the desired behavior — the next GetCredentials call will surface
// AuthRequiredError so the user is prompted to re-authorize.
func TestLocalDirectoryStore_BackwardCompat_BareToken(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewLocalDirectoryStore(dir)
	// Write a bare oauth2.Token JSON shape directly to disk.
	bare := []byte(`{"access_token":"old-style","refresh_token":"r","token_type":"Bearer"}`)
	p := filepath.Join(dir, "legacy%40example.com.json")
	if err := os.WriteFile(p, bare, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("legacy@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Token == nil {
		t.Fatal("want non-nil cred from legacy file")
	}
	if got.Token.AccessToken != "old-style" {
		t.Fatalf("token field mismatch, got %+v", got.Token)
	}
	if len(got.Scopes) != 0 {
		t.Fatalf("legacy file should yield empty Scopes, got %v", got.Scopes)
	}
}

func TestLocalDirectoryStore_StoreNilRejected(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewLocalDirectoryStore(dir)
	if err := store.Store("x@example.com", nil); err == nil {
		t.Fatal("want error for nil cred")
	}
	if err := store.Store("x@example.com", &StoredCredential{Token: nil}); err == nil {
		t.Fatal("want error for nil cred.Token")
	}
}
