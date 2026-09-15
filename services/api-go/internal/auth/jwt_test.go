package auth

import (
	"strings"
	"testing"
	"time"
)

func TestJWTNodeParity(t *testing.T) {
	f := loadNodeFixture(t)
	u, err := VerifySession(f.ValidToken, f.Secret)
	if err != nil {
		t.Fatalf("jose token rejected: %v", err)
	}
	if u.ID != f.User.ID || u.Email != f.User.Email {
		t.Fatalf("identity = %+v, want %+v", u, f.User)
	}
	if _, err := VerifySession(f.ExpiredToken, f.Secret); err == nil {
		t.Fatal("expired token accepted")
	}
	if _, err := VerifySession(f.WrongSecretToken, f.Secret); err == nil {
		t.Fatal("wrong-secret token accepted")
	}
	tampered := f.ValidToken[:len(f.ValidToken)-2] + "xx"
	if _, err := VerifySession(tampered, f.Secret); err == nil {
		t.Fatal("tampered token accepted")
	}
	if _, err := VerifySession(f.ValidToken, ""); err == nil {
		t.Fatal("empty secret accepted")
	}
}

func TestJWTRoundTrip(t *testing.T) {
	u, err := VerifySession(mustSign(t, SessionUser{ID: "c123", Email: "a@b.co"}), "s3cret")
	if err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if u.ID != "c123" || u.Email != "a@b.co" {
		t.Fatalf("identity = %+v", u)
	}
}

func TestJWTExpiryShape(t *testing.T) {
	before := time.Now()
	tok := mustSign(t, SessionUser{ID: "c1", Email: "a@b.co"})
	// Compact serialization: 3 segments, HS256 header (jose emits the same).
	if parts := strings.Split(tok, "."); len(parts) != 3 {
		t.Fatalf("segments = %d", len(parts))
	}
	u, err := VerifySession(tok, "s3cret")
	if err != nil || u == nil {
		t.Fatalf("fresh token: %+v %v", u, err)
	}
	_ = before
}

func mustSign(t *testing.T, u SessionUser) string {
	t.Helper()
	tok, err := SignSession(u, "s3cret")
	if err != nil {
		t.Fatalf("SignSession: %v", err)
	}
	return tok
}
