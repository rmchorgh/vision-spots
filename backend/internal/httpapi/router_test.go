package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rmchorgh/vision-spots/backend/internal/config"
	"github.com/rmchorgh/vision-spots/backend/internal/session"
	"github.com/rmchorgh/vision-spots/backend/internal/spotify"
)

func TestRouter_PublicEndpoints(t *testing.T) {
	cfg := &config.Config{
		SpotifyClientID:     "mock_client_id",
		SpotifyClientSecret: "mock_client_secret",
		SpotifyRedirectURI:  "https://vision-spots.richardmch.org/callback",
		SessionSigningKey:   "signing_key_32_bytes_long_string_for_testing!",
		AllowedOrigin:       "visionspots://callback",
	}
	store := session.NewStore()
	router := NewRouter(cfg, store)

	t.Run("GET /healthz", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/healthz", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		if w.Body.String() != "ok" {
			t.Errorf("expected ok, got %q", w.Body.String())
		}
	})

	t.Run("GET /auth/start", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/auth/start", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		authURL := resp["authorize_url"]
		state := resp["state"]

		if !strings.HasPrefix(authURL, "https://accounts.spotify.com/authorize") {
			t.Errorf("unexpected authorize_url prefix: %s", authURL)
		}
		if state == "" {
			t.Errorf("state should not be empty")
		}
		// State must be independently random — not a prefix of the verifier.
		// Verify the state is stored and retrievable (it's a separate random value).
		verifier, ok := store.GetVerifier(state)
		if !ok || verifier == "" {
			t.Errorf("expected state and verifier to be stored")
		}
		// State and verifier should be distinct values.
		if strings.HasPrefix(verifier, strings.TrimPrefix(state, "st_")) {
			t.Errorf("state appears to be derived from verifier — should be independent")
		}
		// Scope must be percent-encoded in the URL (spaces become + or %%20).
		if strings.Contains(authURL, "scope=user-read-private user-read-email") {
			t.Errorf("scope contains literal spaces; URL encoding is broken")
		}
	})
}

func TestPlaylistTracksHandler(t *testing.T) {
	mockSpotify := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/playlists/pl123/items" {
			limit := 50
			offset := 0
			if v := r.URL.Query().Get("limit"); v != "" {
				limit, _ = strconv.Atoi(v)
			}
			if v := r.URL.Query().Get("offset"); v != "" {
				offset, _ = strconv.Atoi(v)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"total":  3,
				"limit":  limit,
				"offset": offset,
				"items": []map[string]any{
					{
						"track": map[string]any{
							"uri":         "spotify:track:abc",
							"name":        "Song One",
							"duration_ms": 180000,
							"artists":     []map[string]any{{"name": "Artist A"}},
							"album": map[string]any{
								"name":   "Album X",
								"images": []map[string]any{{"url": "https://img/1"}},
							},
						},
					},
					{
						"track": nil, // local file or removed track — should be skipped
					},
					{
						"track": map[string]any{
							"uri":         "spotify:track:def",
							"name":        "Song Two",
							"duration_ms": 240000,
							"artists":     []map[string]any{{"name": "Artist B"}},
							"album": map[string]any{
								"name":   "Album Y",
								"images": []map[string]any{{"url": "https://img/2"}},
							},
						},
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockSpotify.Close()

	oldAPIBaseURL := spotify.APIBaseURL
	spotify.APIBaseURL = mockSpotify.URL
	defer func() { spotify.APIBaseURL = oldAPIBaseURL }()

	cfg := &config.Config{
		SpotifyClientID:     "mock_client_id",
		SpotifyClientSecret: "mock_client_secret",
		SpotifyRedirectURI:  "https://vision-spots.richardmch.org/callback",
		SessionSigningKey:   "signing_key_32_bytes_long_string_for_testing!",
	}
	store := session.NewStore()
	router := NewRouter(cfg, store)

	sessionID := "tracks_session"
	store.SaveTokens(sessionID, session.SpotifyTokens{
		AccessToken:  "valid_token",
		RefreshToken: "refresh_token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	})
	jwtToken, err := session.MintToken(cfg.SessionSigningKey, sessionID, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to mint token: %v", err)
	}

	t.Run("returns tracks with next_offset", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/playlists/pl123/tracks?limit=2&offset=0", nil)
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Items []struct {
				TrackID    string `json:"track_id"`
				Name       string `json:"name"`
				Artist     string `json:"artist"`
				Album      string `json:"album"`
				DurationMS int    `json:"duration_ms"`
				Image      string `json:"image"`
			} `json:"items"`
			Total      int  `json:"total"`
			Limit      int  `json:"limit"`
			Offset     int  `json:"offset"`
			NextOffset *int `json:"next_offset"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// null-track entry must be stripped
		if len(resp.Items) != 2 {
			t.Fatalf("expected 2 items (null stripped), got %d", len(resp.Items))
		}
		if resp.Items[0].TrackID != "spotify:track:abc" {
			t.Errorf("unexpected track_id: %s", resp.Items[0].TrackID)
		}
		if resp.Items[0].Artist != "Artist A" {
			t.Errorf("unexpected artist: %s", resp.Items[0].Artist)
		}
		if resp.Total != 3 {
			t.Errorf("expected total 3, got %d", resp.Total)
		}
		// 0 + 2 items < total 3, so next_offset should be non-nil
		if resp.NextOffset == nil {
			t.Fatal("expected next_offset to be set")
		}
		if *resp.NextOffset != 2 {
			t.Errorf("expected next_offset 2, got %d", *resp.NextOffset)
		}
	})

	t.Run("invalid limit rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/playlists/pl123/tracks?limit=999", nil)
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("requires auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/playlists/pl123/tracks", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})
}

func TestRouter_AuthenticatedAndRefresh(t *testing.T) {
	var callCount int
	mockSpotify := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "new_refreshed_token",
				"refresh_token": "still_the_same_refresh_token",
				"expires_in":    3600,
				"token_type":    "Bearer",
			})
			return
		}

		if r.URL.Path == "/v1/me" {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "Bearer old_expired_token" && callCount == 0 {
				callCount++
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": {"status": 401, "message": "The access token expired"}}`))
				return
			}

			if authHeader == "Bearer new_refreshed_token" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{
					"id":           "spotify_user_123",
					"display_name": "Mock User",
					"product":      "premium",
					"images": []map[string]any{
						{"url": "https://image.url"},
					},
				})
				return
			}

			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("invalid token"))
		}
	}))
	defer mockSpotify.Close()

	oldAccountsURL := spotify.AccountsBaseURL
	spotify.AccountsBaseURL = mockSpotify.URL
	oldAPIBaseURL := spotify.APIBaseURL
	spotify.APIBaseURL = mockSpotify.URL
	defer func() {
		spotify.AccountsBaseURL = oldAccountsURL
		spotify.APIBaseURL = oldAPIBaseURL
	}()

	cfg := &config.Config{
		SpotifyClientID:     "mock_client_id",
		SpotifyClientSecret: "mock_client_secret",
		SpotifyRedirectURI:  "https://vision-spots.richardmch.org/callback",
		SessionSigningKey:   "signing_key_32_bytes_long_string_for_testing!",
		AllowedOrigin:       "visionspots://callback",
	}
	store := session.NewStore()
	router := NewRouter(cfg, store)

	sessionID := "test_session_id"
	store.SaveTokens(sessionID, session.SpotifyTokens{
		AccessToken:  "old_expired_token",
		RefreshToken: "some_refresh_token",
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
	})

	jwtToken, err := session.MintToken(cfg.SessionSigningKey, sessionID, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to mint token: %v", err)
	}

	t.Run("GET /me with 401 Auto-Refresh", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/me", nil)
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var meResp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&meResp); err != nil {
			t.Fatalf("failed to parse /me response: %v", err)
		}

		if meResp["id"] != "spotify_user_123" {
			t.Errorf("expected id %q, got %q", "spotify_user_123", meResp["id"])
		}
		if meResp["display_name"] != "Mock User" {
			t.Errorf("expected display_name %q, got %q", "Mock User", meResp["display_name"])
		}
		if meResp["image"] != "https://image.url" {
			t.Errorf("expected image %q, got %q", "https://image.url", meResp["image"])
		}

		updatedTokens, ok := store.GetTokens(sessionID)
		if !ok {
			t.Fatalf("tokens not found in store")
		}
		if updatedTokens.AccessToken != "new_refreshed_token" {
			t.Errorf("expected AccessToken to be updated to %q, got %q", "new_refreshed_token", updatedTokens.AccessToken)
		}
	})
}
