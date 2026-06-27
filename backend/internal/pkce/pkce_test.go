package pkce

import (
	"encoding/base64"
	"testing"
)

func TestPKCE(t *testing.T) {
	t.Run("GenerateVerifier", func(t *testing.T) {
		verifier, err := GenerateVerifier()
		if err != nil {
			t.Fatalf("unexpected error generating verifier: %v", err)
		}

		if len(verifier) < 43 || len(verifier) > 128 {
			t.Errorf("expected verifier length between 43 and 128, got %d", len(verifier))
		}

		_, err = base64.RawURLEncoding.DecodeString(verifier)
		if err != nil {
			t.Errorf("verifier is not valid RawURLEncoding base64: %v", err)
		}
	})

	t.Run("GenerateChallenge", func(t *testing.T) {
		verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		expectedChallenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

		challenge := GenerateChallenge(verifier)
		if challenge != expectedChallenge {
			t.Errorf("expected challenge %q, got %q", expectedChallenge, challenge)
		}
	})
}
