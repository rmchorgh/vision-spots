package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rmchorgh/vision-spots/backend/internal/config"
	"github.com/rmchorgh/vision-spots/backend/internal/pkce"
	"github.com/rmchorgh/vision-spots/backend/internal/session"
	"github.com/rmchorgh/vision-spots/backend/internal/spotify"
)

// hopByHopHeaders must not be forwarded between HTTP peers per RFC 7230 §6.1.
var hopByHopHeaders = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"Te":                  true,
	"Trailers":            true,
	"Transfer-Encoding":   true,
	"Upgrade":             true,
}

func copyResponseHeaders(dst, src http.Header) {
	for k, v := range src {
		if !hopByHopHeaders[k] {
			for _, val := range v {
				dst.Add(k, val)
			}
		}
	}
}

func NewRouter(cfg *config.Config, store *session.Store) *chi.Mux {
	r := chi.NewRouter()

	r.Get("/healthz", healthHandler)

	if cfg.DebugSpotifyToken != "" {
		r.Post("/debug/mint-session", debugMintSessionHandler(cfg, store))
	}

	r.Get("/auth/start", startAuthHandler(cfg, store))
	r.Get("/callback", callbackHandler(cfg, store))

	r.Group(func(r chi.Router) {
		r.Use(sessionMiddleware(cfg, store))

		r.Get("/me", meHandler(cfg, store))
		r.Post("/auth/refresh", refreshHandler(cfg, store))

		r.HandleFunc("/api/spotify/*", spotifyProxyHandler(cfg, store))

		// Typed playlist endpoint with pagination (avoids pushing cursor construction to the app)
		r.Get("/api/playlists/{playlist_id}/tracks", playlistTracksHandler(cfg, store))

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

func startAuthHandler(cfg *config.Config, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		verifier, err := pkce.GenerateVerifier()
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		challenge := pkce.GenerateChallenge(verifier)

		// Generate state independently from verifier so an observer of the
		// redirect URL cannot infer any bits of the PKCE verifier.
		stateBytes := make([]byte, 16)
		if _, err := rand.Read(stateBytes); err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}
		state := "st_" + hex.EncodeToString(stateBytes)

		store.SaveState(state, verifier, 10*time.Minute)

		// Use url.Values so all parameters are properly percent-encoded,
		// including redirect_uri (which may contain special chars) and scope
		// (whose spaces must be encoded to avoid breaking URL parsing on iOS).
		params := url.Values{}
		params.Set("client_id", cfg.SpotifyClientID)
		params.Set("response_type", "code")
		params.Set("redirect_uri", cfg.SpotifyRedirectURI)
		params.Set("code_challenge", challenge)
		params.Set("code_challenge_method", "S256")
		params.Set("state", state)
		params.Set("scope", "user-read-private user-read-email user-library-read user-library-modify playlist-read-private "+
			"user-read-playback-state user-modify-playback-state user-read-currently-playing user-read-recently-played streaming")

		authURL := "https://accounts.spotify.com/authorize?" + params.Encode()

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

		tokens, err := spotify.ExchangeCode(cfg, code, verifier)
		if err != nil {
			writeError(w, "spotify_error", err.Error(), http.StatusInternalServerError)
			return
		}

		sessionID, err := session.GenerateSessionID()
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		store.SaveTokens(sessionID, session.SpotifyTokens{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			ExpiresAt:    time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second),
		})

		jwtToken, err := session.MintToken(cfg.SessionSigningKey, sessionID, 1*time.Hour)
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		deepLink := "visionspots://callback?session=" + jwtToken

		if isIOSUserAgent(r.UserAgent()) {
			http.Redirect(w, r, deepLink, http.StatusFound)
			return
		}

		webURL := "https://vision-spots.richardmch.org/?session=" + jwtToken
		serveCallbackPage(w, jwtToken, deepLink, webURL)
	}
}

func isIOSUserAgent(ua string) bool {
	ua = strings.ToLower(ua)
	return strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ipod")
}

var callbackPageTmpl = template.Must(template.New("callback").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Vision Spots — Signed In</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      min-height: 100dvh;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      gap: 1.5rem;
      font-family: system-ui, sans-serif;
      background: #0a0a0a;
      color: #f5f5f5;
      padding: 2rem;
    }
    h1 { font-size: 1.5rem; font-weight: 600; }
    p { color: #888; font-size: 0.9rem; }
    .actions { display: flex; flex-direction: column; gap: 0.75rem; width: 100%; max-width: 320px; }
    button, a.btn {
      display: block;
      width: 100%;
      padding: 0.75rem 1.25rem;
      border-radius: 0.5rem;
      font-size: 1rem;
      font-weight: 500;
      text-align: center;
      text-decoration: none;
      cursor: pointer;
      border: none;
      transition: opacity 0.15s;
    }
    button:hover, a.btn:hover { opacity: 0.85; }
    .primary { background: #1db954; color: #000; }
    .secondary { background: #1a1a1a; color: #f5f5f5; border: 1px solid #333; }
    .copied { background: #166534 !important; color: #f5f5f5 !important; }
  </style>
</head>
<body>
  <h1>You&#39;re signed in</h1>
  <p>Head back to the app to continue.</p>
  <div class="actions">
    <button id="copy-btn" class="primary" data-token="{{.Token}}">Copy session token</button>
    <a href="{{.DeepLink}}" class="btn secondary">Open in app</a>
  </div>
  <script>
    document.getElementById('copy-btn').addEventListener('click', function() {
      var btn = this;
      navigator.clipboard.writeText(btn.dataset.token).then(function() {
        btn.textContent = 'Copied!';
        btn.classList.add('copied');
        setTimeout(function() {
          btn.textContent = 'Copy session token';
          btn.classList.remove('copied');
        }, 2000);
      });
    });
  </script>
</body>
</html>`))

func serveCallbackPage(w http.ResponseWriter, token, deepLink, _ string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = callbackPageTmpl.Execute(w, struct {
		Token    string
		DeepLink string
	}{Token: token, DeepLink: deepLink})
}

// executeSpotifyRequest sends req to Spotify with the session's access token.
// On 401, it acquires a per-session lock and refreshes — only the first goroutine
// to reach the lock actually calls Spotify's token endpoint; concurrent goroutines
// wait for the lock and then re-use the already-refreshed token.
func executeSpotifyRequest(cfg *config.Config, store *session.Store, sessionID string, req *http.Request) (*http.Response, error) {
	tokens, ok := store.GetTokens(sessionID)
	if !ok {
		return nil, fmt.Errorf("session tokens not found")
	}

	// Buffer the body so we can replay the request if we need to retry after refresh.
	var bodyBytes []byte
	if req.Body != nil && req.Body != http.NoBody {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	resp, err := spotify.Do(req, tokens.AccessToken)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	resp.Body.Close()

	// Serialize token refresh per session. If two requests hit a 401 at the same
	// time, the second goroutine to acquire the lock will find the token already
	// updated and skip the Spotify call, using the new token for its retry instead.
	mu := store.RefreshLock(sessionID)
	mu.Lock()
	defer mu.Unlock()

	current, ok := store.GetTokens(sessionID)
	if !ok {
		return nil, fmt.Errorf("session tokens not found")
	}
	if current.AccessToken == tokens.AccessToken {
		if current.RefreshToken == "" {
			return nil, fmt.Errorf("spotify token expired and no refresh token is available (debug session?)")
		}
		tr, err := spotify.RefreshToken(cfg, current.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}
		current = session.SpotifyTokens{
			AccessToken:  tr.AccessToken,
			RefreshToken: tr.RefreshToken,
			ExpiresAt:    time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
		}
		store.SaveTokens(sessionID, current)
	}

	if len(bodyBytes) > 0 {
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	return spotify.Do(req, current.AccessToken)
}

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

		store.SaveTokens(sessionID, session.SpotifyTokens{
			AccessToken:  tr.AccessToken,
			RefreshToken: tr.RefreshToken,
			ExpiresAt:    time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
		})

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

		copyResponseHeaders(w.Header(), resp.Header)
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
		writeError(w, "premium_required", "This action requires Spotify Premium", http.StatusForbidden)
		return
	}

	if resp.StatusCode != http.StatusOK {
		writeError(w, "spotify_error", fmt.Sprintf("Spotify returned %d: %s", resp.StatusCode, string(body)), resp.StatusCode)
		return
	}

	copyResponseHeaders(w.Header(), resp.Header)
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

		if resp.StatusCode == http.StatusNoContent {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		handlePlayerResponse(w, resp)
	}
}

type contextKey string

const sessionIDKey contextKey = "sessionID"

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

			_, ok := store.GetTokens(sessionID)
			if !ok {
				writeError(w, "session_expired", "session not found in store", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), sessionIDKey, sessionID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// debugMintSessionHandler creates a session from DEBUG_SPOTIFY_TOKEN and returns a signed
// JWT. Only registered when DEBUG_SPOTIFY_TOKEN is set — never compiled out, but a no-op
// in production where the env var is absent.
func debugMintSessionHandler(cfg *config.Config, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, err := session.GenerateSessionID()
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		store.SaveTokens(sessionID, session.SpotifyTokens{
			AccessToken:  cfg.DebugSpotifyToken,
			RefreshToken: "",
			ExpiresAt:    time.Now().Add(1 * time.Hour),
		})

		jwtToken, err := session.MintToken(cfg.SessionSigningKey, sessionID, 1*time.Hour)
		if err != nil {
			writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]string{"session": jwtToken})
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
