package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSessionCookieShape(t *testing.T) {
	w := httptest.NewRecorder()
	SetSession(w, "tok123", false)
	h := w.Header().Get("Set-Cookie")
	for _, want := range []string{"session=tok123", "Path=/", "HttpOnly", "SameSite=Lax", "Max-Age=604800"} {
		if !strings.Contains(h, want) {
			t.Fatalf("Set-Cookie missing %q: %q", want, h)
		}
	}
	if strings.Contains(h, "Secure") {
		t.Fatalf("dev cookie must not be Secure: %q", h)
	}
	w2 := httptest.NewRecorder()
	SetSession(w2, "tok123", true)
	if !strings.Contains(w2.Header().Get("Set-Cookie"), "Secure") {
		t.Fatal("prod cookie must be Secure")
	}
}

func TestMiddleware(t *testing.T) {
	secret := "mw-secret"
	tok, err := SignSession(SessionUser{ID: "c9", Email: "u@x.co"}, secret)
	if err != nil {
		t.Fatal(err)
	}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UserFromContext(r.Context())
		if !ok || u.ID != "c9" {
			t.Fatalf("context user = %+v %v", u, ok)
		}
		w.WriteHeader(http.StatusOK)
	})
	h := Middleware(secret, inner)

	// No cookie → 401 JSON.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no-cookie status = %d", rec.Code)
	}

	// Valid cookie → passthrough with user.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookie, Value: tok})
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("valid status = %d", rec2.Code)
	}

	// Bad token → 401.
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.AddCookie(&http.Cookie{Name: SessionCookie, Value: "bogus"})
	rec3 := httptest.NewRecorder()
	h.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusUnauthorized {
		t.Fatalf("bogus status = %d", rec3.Code)
	}
}

func TestValidateCredentials(t *testing.T) {
	cases := []struct {
		email, pw string
		ok        bool
	}{
		{"a@b.co", "12345678", true},
		{"not-an-email", "12345678", false},
		{"a@b", "12345678", false},
		{"a@b.co", "short", false},
		{"a@b.co", strings.Repeat("x", 201), false},
	}
	for _, c := range cases {
		err := ValidateCredentials(c.email, c.pw)
		if (err == nil) != c.ok {
			t.Fatalf("Validate(%q, len=%d) = %v", c.email, len(c.pw), err)
		}
	}
}
