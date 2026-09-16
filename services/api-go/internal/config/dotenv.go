package config

import (
	"os"
	"path/filepath"
	"strings"
)

// LoadRootEnv mirrors packages/db loadRootEnv: walk up from cwd (max 6) for
// the single root .env and fill missing process vars (never override). No
// .env found → rely on process env (CI/Docker). Returns the path used, if any.
func LoadRootEnv() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return loadRootEnvFrom(dir)
}

func loadRootEnvFrom(dir string) string {
	for range 6 {
		candidate := filepath.Join(dir, ".env")
		if data, err := os.ReadFile(candidate); err == nil {
			applyDotEnv(string(data))
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
	return ""
}

// applyDotEnv parses KEY=VALUE lines (comments, blanks, `export `, single or
// double quotes) and sets keys absent from the process environment.
func applyDotEnv(data string) {
	for line := range strings.Lines(strings.TrimSpace(data)) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			continue
		}
		val = strings.TrimSpace(val)
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if _, present := os.LookupEnv(key); !present {
			_ = os.Setenv(key, val)
		}
	}
}
