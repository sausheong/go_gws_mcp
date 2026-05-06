package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestLocalDirectoryStore_StoreAndGet(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalDirectoryStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	tok := &oauth2.Token{
		AccessToken:  "atk",
		RefreshToken: "rtk",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour).Round(time.Second),
	}
	if err := store.Store("alice@example.com", tok); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "atk" || got.RefreshToken != "rtk" {
		t.Fatalf("token mismatch: %+v", got)
	}
}

func TestLocalDirectoryStore_GetMissingReturnsNilNoError(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewLocalDirectoryStore(dir)
	tok, err := store.Get("nobody@example.com")
	if err != nil {
		t.Fatalf("want nil err, got %v", err)
	}
	if tok != nil {
		t.Fatal("want nil token for missing user")
	}
}

func TestLocalDirectoryStore_FileMode0600(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewLocalDirectoryStore(dir)
	store.Store("bob@example.com", &oauth2.Token{AccessToken: "x"})

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
	store.Store("c@example.com", &oauth2.Token{AccessToken: "x"})
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
	store.Store("a@example.com", &oauth2.Token{AccessToken: "x"})
	store.Store("b@example.com", &oauth2.Token{AccessToken: "x"})
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
	err := store.Store("../../../etc/passwd", &oauth2.Token{AccessToken: "x"})
	if err == nil {
		t.Fatal("want rejection for traversal-style email")
	}
}
