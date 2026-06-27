package session

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims for our short-lived session token.
type Claims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// Store is v1 in-memory implementation. Single user only.
// Replace with Redis or SQLite for production (see README).
type Store struct {
	states   map[string]StateEntry
	sessions map[string]SpotifyTokens
}

// StateEntry holds the PKCE verifier tied to a state parameter with TTL.
type StateEntry struct {
	Verifier  string
	ExpiresAt time.Time
}

type SpotifyTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func NewStore() *Store {
	return &Store{
		states:   make(map[string]StateEntry),
		sessions: make(map[string]SpotifyTokens),
	}
}

// SaveState stores the PKCE verifier for a given state with a short TTL.
// Called by /auth/start.
func (s *Store) SaveState(state, verifier string, ttl time.Duration) {
	s.states[state] = StateEntry{
		Verifier:  verifier,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// GetVerifier retrieves and deletes the verifier for a state (one-time use).
func (s *Store) GetVerifier(state string) (string, bool) {
	entry, ok := s.states[state]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return "", false
	}
	delete(s.states, state)
	return entry.Verifier, true
}

// SaveTokens stores Spotify tokens against a session ID.
func (s *Store) SaveTokens(sessionID string, tokens SpotifyTokens) {
	s.sessions[sessionID] = tokens
}

// GetTokens retrieves tokens for a session.
func (s *Store) GetTokens(sessionID string) (SpotifyTokens, bool) {
	t, ok := s.sessions[sessionID]
	return t, ok
}

// X: In-memory store for v1. The map is protected by mutex in real version. TTL prevents replay attacks on old states.
