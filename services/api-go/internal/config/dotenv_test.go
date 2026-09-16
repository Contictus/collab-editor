package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyDotEnv(t *testing.T) {
	t.Setenv("DOTENV_KEPT", "orig")
	applyDotEnv(`
# comment
DOTENV_A=plain
DOTENV_B="quoted value"
DOTENV_C='single'
DOTENV_KEPT=changed
export DOTENV_D=exported
EMPTY_OK=
`)
	cases := map[string]string{
		"DOTENV_A":    "plain",
		"DOTENV_B":    "quoted value",
		"DOTENV_C":    "single",
		"DOTENV_KEPT": "orig",
		"DOTENV_D":    "exported",
		"EMPTY_OK":    "",
	}
	for k, want := range cases {
		if got := os.Getenv(k); got != want {
			t.Fatalf("%s = %q, want %q", k, got, want)
		}
	}
}

func TestLoadRootEnvFrom(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("DOTENV_FOUND=yes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := loadRootEnvFrom(nested); got == "" {
		t.Fatal("not found")
	}
	if os.Getenv("DOTENV_FOUND") != "yes" {
		t.Fatal("not applied")
	}
	if got := loadRootEnvFrom(t.TempDir()); got != "" {
		t.Fatalf("false positive: %q", got)
	}
}
