// Package config loads the service environment (F1).
//
// Same contract as the Node side: one DATABASE_URL (root .env, Postgres 16)
// shared by every process, API_PORT for this service. DATABASE_URL is required
// for anything touching the DB; the /health endpoint stays dependency-free.
package config

import (
	"errors"
	"os"
	"strings"
)

// Config is the resolved service environment.
type Config struct {
	// DatabaseURL is the Postgres DSN (same value as the Node apps use).
	DatabaseURL string
	// Port is the HTTP listen port.
	Port string
}

// Load reads the process environment. requireDB controls whether a missing
// DATABASE_URL is an error (migrations, API) or tolerated (bare /health).
func Load(requireDB bool) (Config, error) {
	LoadRootEnv()
	cfg := Config{
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		Port:        strings.TrimSpace(os.Getenv("API_PORT")),
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if requireDB && cfg.DatabaseURL == "" {
		return Config{}, errors.New("config: DATABASE_URL is not set")
	}
	return cfg, nil
}
