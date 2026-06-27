package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims for our short-lived session token.
type Claims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// Store is the v1 in-memory implementation. Thread-safe with a mutex.
type Store struct {
	mu         sync.RWMutex
	states     map[string]StateEntry
	sessions   map[string]SpotifyTokens
	refreshMus sync.Map // map[sessionID]*sync.Mutex — one mutex per session for token refresh
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
func (s *Store) SaveState(state, verifier string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[state] = StateEntry{
		Verifier:  verifier,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// GetVerifier retrieves and deletes the verifier for a state (one-time use).
func (s *Store) GetVerifier(state string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.states[state]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return "", false
	}
	delete(s.states, state)
	return entry.Verifier, true
}

// SaveTokens stores Spotify tokens against a session ID.
func (s *Store) SaveTokens(sessionID string, tokens SpotifyTokens) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = tokens
}

// GetTokens retrieves tokens for a session.
func (s *Store) GetTokens(sessionID string) (SpotifyTokens, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.sessions[sessionID]
	return t, ok
}

// RefreshLock returns the per-session mutex used to serialize token refresh calls.
// Concurrent requests that both receive a 401 should acquire this lock before
// calling the Spotify token endpoint, so only one goroutine actually refreshes.
func (s *Store) RefreshLock(sessionID string) *sync.Mutex {
	mu, _ := s.refreshMus.LoadOrStore(sessionID, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

// GenerateSessionID generates a cryptographically secure random session ID.
func GenerateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// MintToken generates a signed JWT session token for the given session ID.
func MintToken(signingKey string, sessionID string, duration time.Duration) (string, error) {
	claims := Claims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(signingKey))
}

// VerifyToken parses and validates the JWT session token, returning the session ID if valid.
func VerifyToken(signingKey string, tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(signingKey), nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token claims")
	}
	return claims.SessionID, nil
}
