package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// nodeFixture loads testdata/node_parity.json — tokens + PHC produced by the
// real jose and @node-rs/argon2 (see gen script in temp dir; SECRET test-only).
type nodeFixture struct {
	Secret           string `json:"secret"`
	Password         string `json:"password"`
	Argon2idPHC      string `json:"argon2idPHC"`
	User             SessionUser `json:"user"`
	ValidToken       string `json:"validToken"`
	ExpiredToken     string `json:"expiredToken"`
	WrongSecretToken string `json:"wrongSecretToken"`
}

func loadNodeFixture(t *testing.T) nodeFixture {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "node_parity.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var f nodeFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return f
}

func TestPasswordRoundTrip(t *testing.T) {
	h, err := HashPassword("s3cur3-p@ssword!")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("PHC shape = %q", h)
	}
	if !VerifyPassword(h, "s3cur3-p@ssword!") {
		t.Fatal("round-trip verify failed")
	}
	if VerifyPassword(h, "wrong") {
		t.Fatal("wrong password verified")
	}
	// Distinct salts per hash.
	h2, _ := HashPassword("s3cur3-p@ssword!")
	if h == h2 {
		t.Fatal("salts not random")
	}
}

func TestPasswordNodeParity(t *testing.T) {
	f := loadNodeFixture(t)
	if !VerifyPassword(f.Argon2idPHC, f.Password) {
		t.Fatal("node-rs argon2id hash rejected")
	}
	if VerifyPassword(f.Argon2idPHC, "not-the-password") {
		t.Fatal("wrong password verified against node hash")
	}
}

func TestPasswordMalformed(t *testing.T) {
	for _, bad := range []string{
		"", "not-a-hash", "$argon2id$v=19$m=1,t=1,p=1$!!!$???",
		"$argon2i$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA", // wrong variant
		"$argon2id$v=16$m=19456,t=2,p=1$c2FsdA$aGFzaA", // wrong version
	} {
		if VerifyPassword(bad, "anything") {
			t.Fatalf("malformed accepted: %q", bad)
		}
	}
}
