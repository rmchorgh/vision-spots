package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rmchorgh/vision-spots/backend/internal/config"
	"github.com/rmchorgh/vision-spots/backend/internal/pkce"
	"github.com/rmchorgh/vision-spots/backend/internal/session"
	"github.com/rmchorgh/vision-spots/backend/internal/spotify"
)

// NewRouter creates the chi router with all endpoints from api-contract.md.
// Heavily commented so the owner can understand the full OAuth/PKCE/session flow.
func NewRouter(cfg *config.Config, store *session.Store) *chi.Mux {
	r := chi.NewRouter()

	r.Get("/healthz", healthHandler)

	// Phase 1: Public auth endpoints (no Bearer token needed)
	r.Get("/auth/start", startAuthHandler(cfg, store))
	r.Get("/callback", callbackHandler(cfg, store))

	// Phase 2: Protected endpoints (session JWT required)
	r.Group(func(r chi.Router) {
		r.Use(sessionMiddleware(cfg, store))

		r.Get("/me", meHandler(cfg, store))
		r.Post("/auth/refresh", refreshHandler(cfg, store))

		// Generic proxy for Spotify Web API
		r.HandleFunc("/api/spotify/*", spotifyProxyHandler(cfg, store))

		// Typed playlist endpoint with pagination (avoids pushing cursor construction to the app)
		r.Get("/api/playlists/{playlist_id}/tracks", playlistTracksHandler(cfg, store))

		// Spotify Connect player controls
		r.Route("/api/player", func(r chi.Router) {
			r.Get("/devices", playerDevicesHandler(cfg, store))
			r.Put("/play", playerPlayHandler(cfg, store))
			r.Put("/pause", playerPauseHandler(cfg, store))
			r.Post("/next", playerNextHandler(cfg, store))
			r.Post("/previous", playerPreviousHandler(cfg, store))
			r.Get("/state", playerStateHandler(cfg, store))
		})
	})

	return r
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// startAuthHandler begins the PKCE flow.
// Generates verifier + S256 challenge, creates random state, stores the verifier (TODO), returns Spotify authorize URL.
func startAuthHandler(cfg *config.Config, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		verifier, err := pkce.GenerateVerifier()
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		challenge := pkce.GenerateChallenge(verifier)
		state := "st_" + verifier[:12]

		// Store verifier for callback validation (one-time use, short TTL)
		store.SaveState(state, verifier, 10*time.Minute)

		// Build Spotify authorization URL (exact scopes from shared-constants.md)
		authURL := "https://accounts.spotify.com/authorize" +
			"?client_id=" + cfg.SpotifyClientID +
			"&response_type=code" +
			"&redirect_uri=" + cfg.SpotifyRedirectURI +
			"&code_challenge=" + challenge +
			"&code_challenge_method=S256" +
			"&state=" + state +
			"&scope=user-read-private user-read-email user-library-read playlist-read-private " +
			"user-read-playback-state user-modify-playback-state user-read-currently-playing streaming"

		writeJSON(w, map[string]string{
			"authorize_url": authURL,
			"state":         state,
		})
	}
}

func callbackHandler(cfg *config.Config, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if code == "" || state == "" {
			writeError(w, "bad_request", "missing code or state", http.StatusBadRequest)
			return
		}

		verifier, ok := store.GetVerifier(state)
		if !ok {
			writeError(w, "invalid_state", "state expired or invalid", http.StatusBadRequest)
			return
		}

		// Exchange code for tokens using PKCE verifier (client secret stays on server)
		tokens, err := spotify.ExchangeCode(cfg, code, verifier)
		if err != nil {
			writeError(w, "spotify_error", err.Error(), http.StatusInternalServerError)
			return
		}

		// Generate new session ID
		sessionID, err := session.GenerateSessionID()
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		// Store the Spotify tokens in the session store
		spotifyTokens := session.SpotifyTokens{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			ExpiresAt:    time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second),
		}
		store.SaveTokens(sessionID, spotifyTokens)

		// Mint the JWT session token (valid for 1 hour)
		jwtToken, err := session.MintToken(cfg.SessionSigningKey, sessionID, 1*time.Hour)
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		// Redirect to the app deep link
		redirectBase := cfg.AllowedOrigin
		if redirectBase == "" {
			redirectBase = "visionspots://callback"
		}
		http.Redirect(w, r, redirectBase+"?session="+jwtToken, http.StatusFound)
	}
}

// Helper to execute request to Spotify using stored tokens and auto-refresh persistence.
func executeSpotifyRequest(cfg *config.Config, store *session.Store, sessionID string, req *http.Request) (*http.Response, error) {
	tokens, ok := store.GetTokens(sessionID)
	if !ok {
		return nil, fmt.Errorf("session tokens not found")
	}

	resp, tr, err := spotify.Request(cfg, req, tokens.AccessToken, tokens.RefreshToken)
	if err != nil {
		return nil, err
	}

	// If tokens were auto-refreshed, save the updated tokens to store
	if tr != nil {
		spotifyTokens := session.SpotifyTokens{
			AccessToken:  tr.AccessToken,
			RefreshToken: tr.RefreshToken,
			ExpiresAt:    time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
		}
		store.SaveTokens(sessionID, spotifyTokens)
	}

	return resp, nil
}

// meHandler proxies Spotify GET /v1/me to return the current user's profile.
func meHandler(cfg *config.Config, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, ok := r.Context().Value(sessionIDKey).(string)
		if !ok {
			writeError(w, "session_expired", "no active session found", http.StatusUnauthorized)
			return
		}

		req, err := http.NewRequest("GET", spotify.APIBaseURL+"/v1/me", nil)
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		resp, err := executeSpotifyRequest(cfg, store, sessionID, req)
		if err != nil {
			writeError(w, "spotify_error", err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized {
			writeError(w, "session_expired", "Spotify session expired", http.StatusUnauthorized)
			return
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			writeError(w, "spotify_error", fmt.Sprintf("Spotify returned status %d: %s", resp.StatusCode, string(body)), resp.StatusCode)
			return
		}

		var spotifyMe struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			Product     string `json:"product"`
			Images      []struct {
				URL string `json:"url"`
			} `json:"images"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&spotifyMe); err != nil {
			writeError(w, "internal_error", "failed to parse Spotify response", http.StatusInternalServerError)
			return
		}

		imageURL := ""
		if len(spotifyMe.Images) > 0 {
			imageURL = spotifyMe.Images[0].URL
		}

		writeJSON(w, map[string]string{
			"id":           spotifyMe.ID,
			"display_name": spotifyMe.DisplayName,
			"product":      spotifyMe.Product,
			"image":        imageURL,
		})
	}
}

// refreshHandler triggers manual Spotify token refresh and returns a fresh JWT session token.
func refreshHandler(cfg *config.Config, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, ok := r.Context().Value(sessionIDKey).(string)
		if !ok {
			writeError(w, "session_expired", "no active session found", http.StatusUnauthorized)
			return
		}

		tokens, ok := store.GetTokens(sessionID)
		if !ok {
			writeError(w, "session_expired", "session tokens not found", http.StatusUnauthorized)
			return
		}

		tr, err := spotify.RefreshToken(cfg, tokens.RefreshToken)
		if err != nil {
			writeError(w, "spotify_error", err.Error(), http.StatusInternalServerError)
			return
		}

		spotifyTokens := session.SpotifyTokens{
			AccessToken:  tr.AccessToken,
			RefreshToken: tr.RefreshToken,
			ExpiresAt:    time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
		}
		store.SaveTokens(sessionID, spotifyTokens)

		jwtToken, err := session.MintToken(cfg.SessionSigningKey, sessionID, 1*time.Hour)
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]any{
			"session":    jwtToken,
			"expires_in": 3600,
		})
	}
}

// spotifyProxyHandler is a generic transparent proxy for Spotify Web API endpoints.
func spotifyProxyHandler(cfg *config.Config, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, ok := r.Context().Value(sessionIDKey).(string)
		if !ok {
			writeError(w, "session_expired", "no active session found", http.StatusUnauthorized)
			return
		}

		subPath := chi.URLParam(r, "*")
		targetURL := spotify.APIBaseURL + "/v1/" + subPath
		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}

		req, err := http.NewRequest(r.Method, targetURL, r.Body)
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		for k, v := range r.Header {
			if k != "Authorization" {
				for _, val := range v {
					req.Header.Add(k, val)
				}
			}
		}

		resp, err := executeSpotifyRequest(cfg, store, sessionID, req)
		if err != nil {
			writeError(w, "spotify_error", err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		for k, v := range resp.Header {
			for _, val := range v {
				w.Header().Add(k, val)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

// playlistTracksHandler implements GET /api/playlists/{playlist_id}/tracks.
// It wraps Spotify's GET /v1/playlists/{id}/items, reshaping the response into
// a flat, app-friendly shape and computing next_offset so the app never needs
// to construct Spotify cursor URLs.
func playlistTracksHandler(cfg *config.Config, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, ok := r.Context().Value(sessionIDKey).(string)
		if !ok {
			writeError(w, "session_expired", "no active session found", http.StatusUnauthorized)
			return
		}

		playlistID := chi.URLParam(r, "playlist_id")
		if playlistID == "" {
			writeError(w, "bad_request", "playlist_id is required", http.StatusBadRequest)
			return
		}

		// Parse and clamp pagination params
		limit := 50
		offset := 0
		if v := r.URL.Query().Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 || n > 100 {
				writeError(w, "bad_request", "limit must be an integer between 1 and 100", http.StatusBadRequest)
				return
			}
			limit = n
		}
		if v := r.URL.Query().Get("offset"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				writeError(w, "bad_request", "offset must be a non-negative integer", http.StatusBadRequest)
				return
			}
			offset = n
		}

		targetURL := fmt.Sprintf("%s/v1/playlists/%s/items?limit=%d&offset=%d", spotify.APIBaseURL, playlistID, limit, offset)
		req, err := http.NewRequest("GET", targetURL, nil)
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		resp, err := executeSpotifyRequest(cfg, store, sessionID, req)
		if err != nil {
			writeError(w, "spotify_error", err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			writeError(w, "not_found", "playlist not found", http.StatusNotFound)
			return
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			writeError(w, "spotify_error", fmt.Sprintf("Spotify returned %d: %s", resp.StatusCode, string(body)), resp.StatusCode)
			return
		}

		// Spotify's paged items response
		var spotifyResp struct {
			Total  int `json:"total"`
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
			Items  []struct {
				Track *struct {
					URI      string `json:"uri"`
					Name     string `json:"name"`
					Duration int    `json:"duration_ms"`
					Artists  []struct {
						Name string `json:"name"`
					} `json:"artists"`
					Album struct {
						Name   string `json:"name"`
						Images []struct {
							URL string `json:"url"`
						} `json:"images"`
					} `json:"album"`
				} `json:"track"`
			} `json:"items"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&spotifyResp); err != nil {
			writeError(w, "internal_error", "failed to parse Spotify response", http.StatusInternalServerError)
			return
		}

		// Flatten each item into the contract shape; skip null tracks (local files, removed tracks)
		type trackItem struct {
			TrackID    string `json:"track_id"`
			Name       string `json:"name"`
			Artist     string `json:"artist"`
			Album      string `json:"album"`
			DurationMS int    `json:"duration_ms"`
			Image      string `json:"image"`
		}

		items := make([]trackItem, 0, len(spotifyResp.Items))
		for _, it := range spotifyResp.Items {
			if it.Track == nil {
				continue // local files or removed tracks have a null track object
			}
			artist := ""
			if len(it.Track.Artists) > 0 {
				artist = it.Track.Artists[0].Name
			}
			image := ""
			if len(it.Track.Album.Images) > 0 {
				image = it.Track.Album.Images[0].URL
			}
			items = append(items, trackItem{
				TrackID:    it.Track.URI,
				Name:       it.Track.Name,
				Artist:     artist,
				Album:      it.Track.Album.Name,
				DurationMS: it.Track.Duration,
				Image:      image,
			})
		}

		// next_offset is null when we've consumed all items
		var nextOffset *int
		if offset+len(items) < spotifyResp.Total {
			n := offset + limit
			nextOffset = &n
		}

		writeJSON(w, map[string]any{
			"items":       items,
			"total":       spotifyResp.Total,
			"limit":       spotifyResp.Limit,
			"offset":      spotifyResp.Offset,
			"next_offset": nextOffset,
		})
	}
}

func handlePlayerResponse(w http.ResponseWriter, resp *http.Response) {
	if resp.StatusCode == http.StatusNoContent {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusForbidden {
		// Typically means premium is required for playback control
		writeError(w, "premium_required", "This action requires Spotify Premium", http.StatusForbidden)
		return
	}

	if resp.StatusCode != http.StatusOK {
		writeError(w, "spotify_error", fmt.Sprintf("Spotify returned %d: %s", resp.StatusCode, string(body)), resp.StatusCode)
		return
	}

	// Write OK and copy body
	for k, v := range resp.Header {
		for _, val := range v {
			w.Header().Add(k, val)
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func playerDevicesHandler(cfg *config.Config, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, ok := r.Context().Value(sessionIDKey).(string)
		if !ok {
			writeError(w, "session_expired", "no active session found", http.StatusUnauthorized)
			return
		}

		req, err := http.NewRequest("GET", spotify.APIBaseURL+"/v1/me/player/devices", nil)
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		resp, err := executeSpotifyRequest(cfg, store, sessionID, req)
		if err != nil {
			writeError(w, "spotify_error", err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		handlePlayerResponse(w, resp)
	}
}

func playerPlayHandler(cfg *config.Config, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, ok := r.Context().Value(sessionIDKey).(string)
		if !ok {
			writeError(w, "session_expired", "no active session found", http.StatusUnauthorized)
			return
		}

		// Read and parse request body to get device_id and extract other payload
		var payload struct {
			DeviceID   string   `json:"device_id"`
			ContextURI string   `json:"context_uri,omitempty"`
			URIs       []string `json:"uris,omitempty"`
			PositionMS int      `json:"position_ms,omitempty"`
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, "bad_request", "failed to read body", http.StatusBadRequest)
			return
		}

		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &payload); err != nil {
				writeError(w, "bad_request", "invalid JSON payload", http.StatusBadRequest)
				return
			}
		}

		targetURL := spotify.APIBaseURL + "/v1/me/player/play"
		if payload.DeviceID != "" {
			targetURL += "?device_id=" + payload.DeviceID
		}

		// Rebuild the body to send to Spotify
		spotifyPayload := make(map[string]any)
		if payload.ContextURI != "" {
			spotifyPayload["context_uri"] = payload.ContextURI
		}
		if len(payload.URIs) > 0 {
			spotifyPayload["uris"] = payload.URIs
		}
		if payload.PositionMS > 0 {
			spotifyPayload["position_ms"] = payload.PositionMS
		}

		var reqBody io.Reader
		if len(spotifyPayload) > 0 {
			sb, err := json.Marshal(spotifyPayload)
			if err != nil {
				writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
				return
			}
			reqBody = bytes.NewBuffer(sb)
		}

		req, err := http.NewRequest("PUT", targetURL, reqBody)
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}
		if reqBody != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := executeSpotifyRequest(cfg, store, sessionID, req)
		if err != nil {
			writeError(w, "spotify_error", err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		handlePlayerResponse(w, resp)
	}
}

func playerPauseHandler(cfg *config.Config, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, ok := r.Context().Value(sessionIDKey).(string)
		if !ok {
			writeError(w, "session_expired", "no active session found", http.StatusUnauthorized)
			return
		}

		deviceID := r.URL.Query().Get("device_id")
		targetURL := spotify.APIBaseURL + "/v1/me/player/pause"
		if deviceID != "" {
			targetURL += "?device_id=" + deviceID
		}

		req, err := http.NewRequest("PUT", targetURL, nil)
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		resp, err := executeSpotifyRequest(cfg, store, sessionID, req)
		if err != nil {
			writeError(w, "spotify_error", err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		handlePlayerResponse(w, resp)
	}
}

func playerNextHandler(cfg *config.Config, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, ok := r.Context().Value(sessionIDKey).(string)
		if !ok {
			writeError(w, "session_expired", "no active session found", http.StatusUnauthorized)
			return
		}

		deviceID := r.URL.Query().Get("device_id")
		targetURL := spotify.APIBaseURL + "/v1/me/player/next"
		if deviceID != "" {
			targetURL += "?device_id=" + deviceID
		}

		req, err := http.NewRequest("POST", targetURL, nil)
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		resp, err := executeSpotifyRequest(cfg, store, sessionID, req)
		if err != nil {
			writeError(w, "spotify_error", err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		handlePlayerResponse(w, resp)
	}
}

func playerPreviousHandler(cfg *config.Config, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, ok := r.Context().Value(sessionIDKey).(string)
		if !ok {
			writeError(w, "session_expired", "no active session found", http.StatusUnauthorized)
			return
		}

		deviceID := r.URL.Query().Get("device_id")
		targetURL := spotify.APIBaseURL + "/v1/me/player/previous"
		if deviceID != "" {
			targetURL += "?device_id=" + deviceID
		}

		req, err := http.NewRequest("POST", targetURL, nil)
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		resp, err := executeSpotifyRequest(cfg, store, sessionID, req)
		if err != nil {
			writeError(w, "spotify_error", err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		handlePlayerResponse(w, resp)
	}
}

func playerStateHandler(cfg *config.Config, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, ok := r.Context().Value(sessionIDKey).(string)
		if !ok {
			writeError(w, "session_expired", "no active session found", http.StatusUnauthorized)
			return
		}

		req, err := http.NewRequest("GET", spotify.APIBaseURL+"/v1/me/player", nil)
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		resp, err := executeSpotifyRequest(cfg, store, sessionID, req)
		if err != nil {
			writeError(w, "spotify_error", err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// If Spotify returns 204 No Content for player state (meaning no active playback), we return 204
		if resp.StatusCode == http.StatusNoContent {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		handlePlayerResponse(w, resp)
	}
}

type contextKey string

const (
	sessionIDKey contextKey = "sessionID"
)

func sessionMiddleware(cfg *config.Config, store *session.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
				writeError(w, "session_expired", "Authorization Bearer token required", http.StatusUnauthorized)
				return
			}
			tokenStr := authHeader[7:]

			sessionID, err := session.VerifyToken(cfg.SessionSigningKey, tokenStr)
			if err != nil {
				writeError(w, "session_expired", "invalid or expired session token", http.StatusUnauthorized)
				return
			}

			// Ensure the session actually exists in our store
			_, ok := store.GetTokens(sessionID)
			if !ok {
				writeError(w, "session_expired", "session not found in store", http.StatusUnauthorized)
				return
			}

			// Attach sessionID to context
			ctx := context.WithValue(r.Context(), sessionIDKey, sessionID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func errorHandler(code string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, code, "endpoint not yet implemented", 501)
	}
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}
