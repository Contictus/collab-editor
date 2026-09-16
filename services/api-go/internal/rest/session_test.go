package rest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLogoutAndMe(t *testing.T) {
	api := testAPI(t)
	email := "me-" + time.Now().Format("150405.000000000") + "@test.local"
	_, cookies := testUser(t, api, email)
	var session *http.Cookie
	for _, c := range cookies {
		if c.Name == "session" {
			session = c
		}
	}

	// me with cookie → identity.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	r.AddCookie(session)
	api.handleMe(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), email) {
		t.Fatalf("me = %d %s", w.Code, w.Body.String())
	}

	// me without → 401.
	w2 := httptest.NewRecorder()
	api.handleMe(w2, httptest.NewRequest(http.MethodGet, "/api/auth/me", nil))
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("anon me = %d", w2.Code)
	}

	// logout clears the cookie with 204.
	w3 := httptest.NewRecorder()
	api.handleLogout(w3, httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil))
	if w3.Code != http.StatusNoContent {
		t.Fatalf("logout = %d", w3.Code)
	}
	setCookie := w3.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "session=") || !strings.Contains(setCookie, "Max-Age=0") {
		t.Fatalf("clear = %q", setCookie)
	}
}
