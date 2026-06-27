package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rmchorgh/vision-spots/backend/internal/config"
	"github.com/rmchorgh/vision-spots/backend/internal/pkce"
	"github.com/rmchorgh/vision-spots/backend/internal/session"
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

		r.Get("/me", meHandler(store))
		r.Post("/auth/refresh", refreshHandler(store))

		// Generic proxy for Spotify Web API
		r.HandleFunc("/api/spotify/*", spotifyProxyHandler(store))

		// Spotify Connect player controls
		r.Route("/api/player", func(r chi.Router) {
			r.Get("/devices", playerDevicesHandler(store))
			r.Put("/play", playerPlayHandler(store))
			r.Put("/pause", playerPauseHandler(store))
			r.Post("/next", playerNextHandler(store))
			r.Post("/previous", playerPreviousHandler(store))
			r.Get("/state", playerStateHandler(store))
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
		state := "st_" + verifier[:12] // simple state for v1 (improve with crypto/rand later)

		// TODO: store state -> verifier with short TTL in session.Store
		// store.SaveState(state, verifier, time.Minute*10)

		// Build Spotify authorization URL exactly per shared constants and contract
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
		// Full implementation pending: 
		// 1. Validate state matches stored verifier
		// 2. Exchange code for access/refresh token using Spotify client (with client_secret)
		// 3. Create session, mint signed JWT
		// 4. 302 redirect to visionspots://callback?session=eyJ...
		http.Redirect(w, r, cfg.AllowedOrigin+"?error=callback_not_fully_implemented_yet", http.StatusFound)
	}
}

// Placeholder handlers (full versions will call Spotify client with token from session)
func meHandler(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{
			"status":  "ok",
			"message": "TODO: proxy to Spotify /v1/me",
		})
	}
}

func refreshHandler(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, "not_implemented", "token refresh not yet implemented", 501)
	}
}

func spotifyProxyHandler(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, "not_implemented", "Spotify proxy not yet implemented", 501)
	}
}

func playerDevicesHandler(s *session.Store) http.HandlerFunc { return errorHandler("not_implemented") }
func playerPlayHandler(s *session.Store) http.HandlerFunc    { return errorHandler("not_implemented") }
func playerPauseHandler(s *session.Store) http.HandlerFunc   { return errorHandler("not_implemented") }
func playerNextHandler(s *session.Store) http.HandlerFunc    { return errorHandler("not_implemented") }
func playerPreviousHandler(s *session.Store) http.HandlerFunc { return errorHandler("not_implemented") }
func playerStateHandler(s *session.Store) http.HandlerFunc   { return errorHandler("not_implemented") }

func sessionMiddleware(cfg *config.Config, store *session.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: extract Bearer token, validate JWT signature with SESSION_SIGNING_KEY,
			// load session from store, auto-refresh Spotify token on 401, attach to context.
			next.ServeHTTP(w, r)
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
