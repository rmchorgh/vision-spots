package session

import (
	"testing"
	"time"
)

func TestStore_StateTTL(t *testing.T) {
	store := NewStore()

	// 1. Save and retrieve valid state
	store.SaveState("state1", "verifier1", 10*time.Minute)
	v, ok := store.GetVerifier("state1")
	if !ok {
		t.Fatalf("expected to get verifier for state1")
	}
	if v != "verifier1" {
		t.Errorf("expected verifier1, got %s", v)
	}

	// 2. Ensure state is one-time use (deleted upon retrieval)
	_, ok = store.GetVerifier("state1")
	if ok {
		t.Errorf("expected state1 to be deleted after first retrieval")
	}

	// 3. Expiration TTL test
	store.SaveState("state2", "verifier2", -1*time.Second) // expired
	_, ok = store.GetVerifier("state2")
	if ok {
		t.Errorf("expected state2 to be expired and not retrieved")
	}
}

func TestStore_Tokens(t *testing.T) {
	store := NewStore()

	tokens := SpotifyTokens{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	store.SaveTokens("session1", tokens)

	retrieved, ok := store.GetTokens("session1")
	if !ok {
		t.Fatalf("expected to retrieve tokens for session1")
	}

	if retrieved.AccessToken != tokens.AccessToken || retrieved.RefreshToken != tokens.RefreshToken {
		t.Errorf("retrieved tokens do not match saved tokens")
	}
}

func TestJWT_MintAndVerify(t *testing.T) {
	signingKey := "super_secret_signing_key_32_bytes_long_string!"
	sessionID := "session_abc_123"

	// Mint token
	token, err := MintToken(signingKey, sessionID, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error minting token: %v", err)
	}

	// Verify token
	parsedSessionID, err := VerifyToken(signingKey, token)
	if err != nil {
		t.Fatalf("unexpected error verifying token: %v", err)
	}

	if parsedSessionID != sessionID {
		t.Errorf("expected session ID %q, got %q", sessionID, parsedSessionID)
	}

	// Verify expired token
	expiredToken, err := MintToken(signingKey, sessionID, -5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error minting expired token: %v", err)
	}

	_, err = VerifyToken(signingKey, expiredToken)
	if err == nil {
		t.Errorf("expected error verifying expired token, but got nil")
	}
}
