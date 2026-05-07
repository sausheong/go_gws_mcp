package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"
)

// StoredCredential pairs an oauth2 token with the OAuth scopes that were
// actually granted at issue time. Storing the granted set lets GetCredentials
// detect tokens that lack scopes a tool now requires (e.g. an earlier auth
// flow ran with fewer scopes, or the user un-checked some on consent).
type StoredCredential struct {
	Token  *oauth2.Token `json:"token"`
	Scopes []string      `json:"scopes,omitempty"`
}

// CredentialStore is the interface for OAuth credential persistence.
// Implementations: LocalDirectoryStore (file-based). Future: GCS, Valkey.
type CredentialStore interface {
	Get(email string) (*StoredCredential, error)
	Store(email string, cred *StoredCredential) error
	Delete(email string) error
	List() ([]string, error)
}

// LocalDirectoryStore writes one URL-encoded JSON file per user under baseDir.
type LocalDirectoryStore struct {
	baseDir string
}

// NewLocalDirectoryStore ensures baseDir exists with mode 0700 and returns a store.
func NewLocalDirectoryStore(baseDir string) (*LocalDirectoryStore, error) {
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, fmt.Errorf("create credentials dir: %w", err)
	}
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}
	return &LocalDirectoryStore{baseDir: abs}, nil
}

// validateEmail rejects empty strings and obvious path-traversal patterns.
// Real email addresses cannot contain '/', '\\', or start with '.'.
func validateEmail(email string) error {
	if strings.TrimSpace(email) == "" {
		return errors.New("email must not be empty")
	}
	if strings.ContainsAny(email, `/\`) {
		return fmt.Errorf("invalid email %q: contains path separator", email)
	}
	if strings.HasPrefix(email, ".") {
		return fmt.Errorf("invalid email %q: starts with dot", email)
	}
	return nil
}

func (s *LocalDirectoryStore) credPath(email string) (string, error) {
	if err := validateEmail(email); err != nil {
		return "", err
	}
	safe := url.QueryEscape(email)
	p := filepath.Join(s.baseDir, safe+".json")
	resolved, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(resolved, s.baseDir+string(os.PathSeparator)) && resolved != s.baseDir {
		return "", fmt.Errorf("invalid credential path: %q", resolved)
	}
	return resolved, nil
}

func (s *LocalDirectoryStore) Get(email string) (*StoredCredential, error) {
	p, err := s.credPath(email)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var cred StoredCredential
	if err := json.Unmarshal(data, &cred); err != nil {
		return nil, fmt.Errorf("parse cred for %s: %w", email, err)
	}
	if cred.Token != nil {
		return &cred, nil
	}
	// Backward compat: older files were a bare oauth2.Token. Try that shape.
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("parse token for %s: %w", email, err)
	}
	if tok.AccessToken == "" && tok.RefreshToken == "" {
		return nil, fmt.Errorf("empty credential file for %s", email)
	}
	return &StoredCredential{Token: &tok}, nil // empty Scopes; caller will trigger re-auth
}

func (s *LocalDirectoryStore) Store(email string, cred *StoredCredential) error {
	if cred == nil || cred.Token == nil {
		return errors.New("cred and cred.Token must be non-nil")
	}
	p, err := s.credPath(email)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (s *LocalDirectoryStore) Delete(email string) error {
	p, err := s.credPath(email)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *LocalDirectoryStore) List() ([]string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		stem := strings.TrimSuffix(name, ".json")
		decoded, err := url.QueryUnescape(stem)
		if err != nil {
			continue
		}
		if !strings.Contains(decoded, "@") {
			continue
		}
		out = append(out, decoded)
	}
	return out, nil
}
