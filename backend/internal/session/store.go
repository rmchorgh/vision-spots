package session

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims for our short-lived session token.
// The app only ever holds this JWT — never raw Spotify tokens.
type Claims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// Store is the v1 in-memory token store (state → verifier, sessionID → spotify tokens).
// Note in README: replace this map+mutex with Redis or SQLite for multi-user / persistence.
type Store struct {
	states   map[string]StateEntry // state → verifier + expiry
	sessions map[string]SpotifyTokens
	// TODO: add mutex for production
}

// StateEntry holds PKCE verifier with short TTL.
type StateEntry struct {
	Verifier  string
	ExpiresAt time.Time
}

type SpotifyTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// NewStore creates the in-memory store.
func NewStore() *Store {
	return &Store{
		states:   make(map[string]StateEntry),
		sessions: make(map[string]SpotifyTokens),
	}
}

// X: The backend owns the real Spotify tokens. The visionOS app only receives a signed session JWT.
// This design keeps the client secret and long-lived refresh tokens server-side.
