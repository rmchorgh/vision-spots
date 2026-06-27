package spotify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/rmchorgh/vision-spots/backend/internal/config"
)

// AccountsBaseURL can be overridden during tests to point to a mock server.
var AccountsBaseURL = "https://accounts.spotify.com"

// APIBaseURL can be overridden during tests to point to a mock server.
var APIBaseURL = "https://api.spotify.com"

// TokenResponse from Spotify token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// ExchangeCode performs the PKCE token exchange.
func ExchangeCode(cfg *config.Config, code, verifier string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", cfg.SpotifyRedirectURI)
	data.Set("client_id", cfg.SpotifyClientID)
	data.Set("code_verifier", verifier)

	req, err := http.NewRequest("POST", AccountsBaseURL+"/api/token", bytes.NewBufferString(data.Encode()))
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

// RefreshToken exchanges a refresh token for a new access token.
// If Spotify does not return a new refresh token, the old one is reused.
func RefreshToken(cfg *config.Config, refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", cfg.SpotifyClientID)
	data.Set("client_secret", cfg.SpotifyClientSecret)

	req, err := http.NewRequest("POST", AccountsBaseURL+"/api/token", bytes.NewBufferString(data.Encode()))
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
		return nil, fmt.Errorf("token refresh failed: %s", string(body))
	}

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, err
	}

	if tr.RefreshToken == "" {
		tr.RefreshToken = refreshToken
	}
	return &tr, nil
}

// Do executes req against Spotify with the given access token and returns the raw response.
// The caller is responsible for handling non-2xx status codes, including 401s.
func Do(req *http.Request, accessToken string) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+accessToken)
	return http.DefaultClient.Do(req)
}
