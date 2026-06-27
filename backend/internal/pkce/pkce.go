package pkce

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// GenerateVerifier creates a high-entropy code_verifier for PKCE.
// Must be 43-128 characters, using unreserved chars [A-Z]/[a-z]/[0-9]/-/. /_/~.
func GenerateVerifier() (string, error) {
	// 32 bytes = 256 bits of entropy is standard and sufficient
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random verifier: %w", err)
	}
	// Base64 URL encoding without padding (per RFC 7636)
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateChallenge creates the code_challenge using S256 method.
// This is SHA256(verifier) then base64url encoded.
// Spotify requires S256 (not plain).
func GenerateChallenge(verifier string) string {
	h := sha256.New()
	h.Write([]byte(verifier))
	sum := h.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(sum)
}

// X: This is the core of modern OAuth security. The app never sends the verifier to Spotify until
// the callback. The backend stores the verifier (tied to 'state') and only uses it during token exchange.
// This prevents authorization code interception attacks.
