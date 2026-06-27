package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/rmchorgh/vision-spots/backend/internal/shared"
)

// Config holds all settings loaded from environment.
type Config struct {
	Port                int
	SpotifyClientID     string
	SpotifyClientSecret string
	SpotifyRedirectURI  string
	SessionSigningKey   string
	AllowedOrigin       string
}

// Load reads environment variables and validates required fields.
func Load() (*Config, error) {
	c := &Config{
		SpotifyClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		SpotifyClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		SpotifyRedirectURI:  os.Getenv("SPOTIFY_REDIRECT_URI"),
		SessionSigningKey:   os.Getenv("SESSION_SIGNING_KEY"),
		AllowedOrigin:       os.Getenv("ALLOWED_ORIGIN"),
	}

	portStr := os.Getenv("PORT")
	if portStr == "" {
		portStr = "5055"
	}
	var err error
	c.Port, err = strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}

	if c.SpotifyClientID == "" {
		return nil, fmt.Errorf("SPOTIFY_CLIENT_ID is required (register app at developer.spotify.com)")
	}
	if c.SpotifyClientSecret == "" {
		return nil, fmt.Errorf("SPOTIFY_CLIENT_SECRET is required")
	}
	if c.SessionSigningKey == "" {
		return nil, fmt.Errorf("SESSION_SIGNING_KEY is required (generate 32+ random bytes, base64)")
	}
	if c.SpotifyRedirectURI == "" {
		c.SpotifyRedirectURI = shared.DefaultRedirectURI
	}

	return c, nil
}
