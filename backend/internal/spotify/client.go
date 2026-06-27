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

// ExchangeCode performs the PKCE token exchange using the verifier from state store.
// This is the critical step where the client_secret is used (never sent to the app).
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

// RefreshToken performs a token refresh request to Spotify using the client_secret and the stored refresh_token.
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

	// If Spotify does not return a new refresh token, we reuse the old one.
	if tr.RefreshToken == "" {
		tr.RefreshToken = refreshToken
	}

	return &tr, nil
}

// Request executes an HTTP request to Spotify, adding the Bearer token.
// If a 401 is received, it uses the provided refresh token to get a new access token,
// updates the request, and retries the request exactly once.
// If refreshed, the new TokenResponse is returned along with the http.Response so the caller can save it.
func Request(cfg *config.Config, req *http.Request, accessToken, refreshToken string) (*http.Response, *TokenResponse, error) {
	// If there is a body, we read and cache it so we can retry if needed
	var bodyBytes []byte
	if req.Body != nil && req.Body != http.NoBody {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, nil, err
		}
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized && refreshToken != "" {
		resp.Body.Close()

		// Attempt to refresh
		tr, err := RefreshToken(cfg, refreshToken)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to auto-refresh token: %w", err)
		}

		// Re-create the request body if it existed
		if len(bodyBytes) > 0 {
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
		req.Header.Set("Authorization", "Bearer "+tr.AccessToken)

		respRetry, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, nil, err
		}
		return respRetry, tr, nil
	}

	return resp, nil, nil
}

// X: This is where the magic happens. The backend uses the client_secret (safe here) to exchange the one-time code + PKCE verifier for real tokens. The app never sees the refresh_token.
