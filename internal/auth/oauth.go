package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"sync"
)

// generatePKCE returns (verifier, S256-challenge, error) per RFC 7636.
func generatePKCE() (verifier, challenge string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

// stateStore is a process-local map of OAuth state -> PKCE verifier.
// State is single-use; Consume removes the entry.
type stateStore struct {
	mu sync.Mutex
	m  map[string]string
}

func newStateStore() *stateStore {
	return &stateStore{m: make(map[string]string)}
}

func (s *stateStore) Store(state, verifier string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[state] = verifier
}

func (s *stateStore) Consume(state string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.m[state]
	if ok {
		delete(s.m, state)
	}
	return v, ok
}

// globalStateStore is the package-level singleton used by StartAuthFlow / HandleAuthCallback.
var globalStateStore = newStateStore()
