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
	"os"
)

// BaseURL is the backend address. Override with the BASE_URL environment variable.
var BaseURL = func() string {
	if u := os.Getenv("BASE_URL"); u != "" {
		return u
	}
	return "http://localhost:5055"
}()

// MustGetToken obtains a session JWT from the local backend and panics if it
// fails. Intended for use in test setup (TestMain or t.Helper wrappers).
func MustGetToken() string {
	token, err := GetToken()
	if err != nil {
		panic(fmt.Sprintf("local_testing: %v", err))
	}
	return token
}

// GetToken returns a session JWT for use in tests. If SESSION_JWT is set in
// the environment it is returned directly; otherwise it POSTs to
// /debug/mint-session on the local backend (only active when
// DEBUG_SPOTIFY_TOKEN is set in the backend's environment).
func GetToken() (string, error) {
	if tok := os.Getenv("SESSION_JWT"); tok != "" {
		return tok, nil
	}
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
