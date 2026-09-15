package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("API_PORT", "")
	t.Setenv("DATABASE_URL", "")
	cfg, err := Load(false)
	if err != nil {
		t.Fatalf("Load(false) = %v", err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want 8080", cfg.Port)
	}
}

func TestLoadRequiresDB(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(true); err == nil {
		t.Fatal("Load(true) without DATABASE_URL should fail")
	}
	t.Setenv("DATABASE_URL", "postgresql://collab:collab@localhost:5432/collab?schema=public")
	cfg, err := Load(true)
	if err != nil {
		t.Fatalf("Load(true) with DATABASE_URL = %v", err)
	}
	if cfg.DatabaseURL == "" {
		t.Fatal("DatabaseURL empty after Load")
	}
}
