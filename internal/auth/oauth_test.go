package auth

import (
	"strings"
	"testing"
)

func TestGeneratePKCE_ProducesS256Pair(t *testing.T) {
	verifier, challenge, err := generatePKCE()
	if err != nil {
		t.Fatal(err)
	}
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Errorf("verifier length %d out of RFC 7636 range", len(verifier))
	}
	if challenge == verifier {
		t.Error("challenge must not equal verifier (S256 must hash)")
	}
	if strings.ContainsAny(challenge, "+/=") {
		t.Errorf("challenge contains non-base64url chars: %s", challenge)
	}
}

func TestStateStore_StoreAndConsume(t *testing.T) {
	store := newStateStore()
	store.Store("state-abc", "verifier-xyz")
	v, ok := store.Consume("state-abc")
	if !ok || v != "verifier-xyz" {
		t.Fatalf("got (%q, %v), want (verifier-xyz, true)", v, ok)
	}
	// Second consume should fail (single-use).
	if _, ok := store.Consume("state-abc"); ok {
		t.Fatal("state should be single-use")
	}
}

func TestStateStore_ConsumeMissing(t *testing.T) {
	store := newStateStore()
	if _, ok := store.Consume("never-stored"); ok {
		t.Fatal("missing state should return ok=false")
	}
}
