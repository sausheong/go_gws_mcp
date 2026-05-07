package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// scopesFromToken extracts the granted scopes from a token's `scope` extra
// (Google's OAuth2 token response includes a space-separated `scope` field).
// Returns nil when no `scope` field is present.
func scopesFromToken(t *oauth2.Token) []string {
	if t == nil {
		return nil
	}
	if v, ok := t.Extra("scope").(string); ok && v != "" {
		return strings.Fields(v)
	}
	return nil
}

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

// AuthRequiredError signals the caller (typically a tool wrapper) that the
// user must complete an OAuth flow. Its Error() returns the LLM-targeted prose.
type AuthRequiredError struct {
	Message string
	AuthURL string
}

func (e *AuthRequiredError) Error() string { return e.Message }

// OAuthClient holds the configured OAuth client + dependencies needed by the flow.
type OAuthClient struct {
	Config       *oauth2.Config
	Store        CredentialStore
	StatePersist string // optional: path to oauth_states.json (skeleton: in-memory only)
}

// NewOAuthClient builds an oauth2.Config from server config + scopes.
func NewOAuthClient(clientID, clientSecret, redirectURI string, scopes []string, store CredentialStore) *OAuthClient {
	return &OAuthClient{
		Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURI,
			Scopes:       scopes,
			Endpoint:     google.Endpoint,
		},
		Store: store,
	}
}

// StartAuthFlow returns an LLM-targeted "ACTION REQUIRED" message containing
// the auth URL the user must visit. Stores PKCE state for callback validation.
func (c *OAuthClient) StartAuthFlow(userEmail, serviceName string) (*AuthRequiredError, error) {
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	verifier, challenge, err := generatePKCE()
	if err != nil {
		return nil, err
	}
	globalStateStore.Store(state, verifier)

	authURL := c.Config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	display := serviceName
	if userEmail != "" {
		display = fmt.Sprintf("%s for '%s'", serviceName, userEmail)
	}

	msg := fmt.Sprintf(`**ACTION REQUIRED: Google Authentication Needed for %s**

To proceed, the user must authorize this application for %s access using all required permissions.
**LLM, please present this exact authorization URL to the user as a clickable hyperlink:**
Authorization URL: %s
Markdown for hyperlink: [Click here to authorize %s access](%s)

**LLM, after presenting the link, instruct the user as follows:**
1. Click the link and complete the authorization in their browser.
2. After successful authorization, **retry their original command**.

The application will use the new credentials. If '%s' was provided, it must match the authenticated account.`,
		display, serviceName, authURL, serviceName, authURL, userEmail)

	return &AuthRequiredError{Message: msg, AuthURL: authURL}, nil
}

// HandleAuthCallback exchanges the authorization code for tokens, fetches the
// authenticated user's email, persists the token, and returns (email, nil).
func (c *OAuthClient) HandleAuthCallback(ctx context.Context, state, code string) (string, error) {
	verifier, ok := globalStateStore.Consume(state)
	if !ok {
		return "", errors.New("invalid or expired OAuth state")
	}

	token, err := c.Config.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", verifier),
	)
	if err != nil {
		return "", fmt.Errorf("token exchange: %w", err)
	}

	email, err := fetchUserEmail(ctx, c.Config, token)
	if err != nil {
		return "", fmt.Errorf("fetch user email: %w", err)
	}

	grantedScopes := scopesFromToken(token)
	if len(grantedScopes) == 0 {
		// Token response didn't include a scope field; fall back to what we
		// requested. (Google always echoes `scope` for new tokens, so this
		// branch is mostly defensive.)
		grantedScopes = c.Config.Scopes
	}
	if err := c.Store.Store(email, &StoredCredential{Token: token, Scopes: grantedScopes}); err != nil {
		return "", fmt.Errorf("persist token: %w", err)
	}
	return email, nil
}

// GetCredentials returns a refreshed *oauth2.Token for userEmail with at least
// the requested scopes (hierarchy-aware). Returns *AuthRequiredError when the
// user needs to authenticate.
func (c *OAuthClient) GetCredentials(ctx context.Context, userEmail string, requiredScopes []string) (*oauth2.Token, error) {
	if userEmail == "" {
		return nil, errors.New("user_google_email is required")
	}
	stored, err := c.Store.Get(userEmail)
	if err != nil {
		return nil, err
	}
	if stored == nil {
		return nil, &AuthRequiredError{Message: fmt.Sprintf("No credentials for %s; user must authenticate", userEmail)}
	}

	// Refresh if needed via TokenSource (auto-refreshes when expired).
	src := c.Config.TokenSource(ctx, stored.Token)
	fresh, err := src.Token()
	if err != nil {
		// Refresh failed (revoked / expired beyond refresh).
		_ = c.Store.Delete(userEmail)
		return nil, &AuthRequiredError{Message: fmt.Sprintf("Credentials for %s expired or revoked; user must re-authenticate. Underlying error: %v", userEmail, err)}
	}

	// If the refresh response carried a fresh `scope` field, prefer it;
	// otherwise keep what we last persisted.
	grantedScopes := stored.Scopes
	if newScopes := scopesFromToken(fresh); len(newScopes) > 0 {
		grantedScopes = newScopes
	}

	if fresh.AccessToken != stored.Token.AccessToken {
		_ = c.Store.Store(userEmail, &StoredCredential{Token: fresh, Scopes: grantedScopes})
	}

	// Check granted scopes (not currently-configured scopes). This catches
	// tokens minted in an earlier run with fewer scopes, or tokens where the
	// user un-checked some scopes on the consent screen.
	if !HasRequiredScopes(grantedScopes, requiredScopes) {
		return nil, &AuthRequiredError{
			Message: fmt.Sprintf("Token for %s lacks required scopes (granted=%d, needed=%d); user must re-authorize", userEmail, len(grantedScopes), len(requiredScopes)),
		}
	}
	return fresh, nil
}

func fetchUserEmail(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token) (string, error) {
	client := cfg.Client(ctx, token)
	resp, err := client.Get("https://openidconnect.googleapis.com/v1/userinfo")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("userinfo returned %d", resp.StatusCode)
	}
	var info struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", err
	}
	if info.Email == "" {
		return "", errors.New("userinfo response missing email")
	}
	return info.Email, nil
}
