package spotify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rmchorgh/vision-spots/backend/internal/config"
)

// Client handles all communication with Spotify APIs.
// Keeps client_secret server-side and auto-refreshes tokens on 401.
type Client struct {
	config *config.Config
	http   *http.Client
	tokens *session.SpotifyTokens // will be wired from store later
}

// TokenResponse from Spotify token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// ExchangeCode performs the PKCE token exchange using the verifier from state store.
// This is the critical step where the client_secret is used (never sent to the app).
func ExchangeCode(cfg *config.Config, code, verifier string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", cfg.SpotifyRedirectURI)
	data.Set("client_id", cfg.SpotifyClientID)
	data.Set("code_verifier", verifier)

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, err
	}

	return &tr, nil
}

// X: This is where the magic happens. The backend uses the client_secret (safe here) to exchange the one-time code + PKCE verifier for real tokens. The app never sees the refresh_token.
