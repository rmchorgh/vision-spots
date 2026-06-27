//go:build darwin

// Package local_testing provides helpers for running integration tests against
// a local backend using a live session JWT.
//
// Set DEBUG_SPOTIFY_TOKEN in your .env (or shell) and start the backend with
// make run. MustGetToken will POST to /debug/mint-session and return a signed
// session JWT.
package local_testing

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	// BaseURL is the local backend address.
	BaseURL = "http://localhost:5055"
)

// MustGetToken obtains a session JWT from the local backend and panics if it
// fails. Intended for use in test setup (TestMain or t.Helper wrappers).
func MustGetToken() string {
	token, err := GetToken()
	if err != nil {
		panic(fmt.Sprintf("local_testing: %v", err))
	}
	return token
}

// GetToken POSTs to /debug/mint-session on the local backend, which is only
// active when DEBUG_SPOTIFY_TOKEN is set in the backend's environment.
func GetToken() (string, error) {
	resp, err := http.Post(BaseURL+"/debug/mint-session", "application/json", nil)
	if err != nil {
		return "", fmt.Errorf("POST /debug/mint-session failed — is the backend running? (make run): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("debug endpoint not found — set DEBUG_SPOTIFY_TOKEN in backend/.env and restart the backend")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("POST /debug/mint-session returned %d", resp.StatusCode)
	}

	var body struct {
		Session string `json:"session"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("decode mint-session response: %w", err)
	}
	if body.Session == "" {
		return "", fmt.Errorf("mint-session returned empty session")
	}
	return body.Session, nil
}
