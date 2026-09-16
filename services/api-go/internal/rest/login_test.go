package rest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func login(t *testing.T, api *API, email, password string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login",
		strings.NewReader(`{"email":"`+email+`","password":"`+password+`"}`))
	api.handleLogin(w, r)
	return w
}

func TestLogin(t *testing.T) {
	api := testAPI(t)
	email := "login-" + time.Now().Format("150405.000000000") + "@test.local"
	testUser(t, api, email)

	w := login(t, api, email, "password123")
	if w.Code != http.StatusOK {
		t.Fatalf("login = %d %s", w.Code, w.Body.String())
	}
	if len(w.Result().Cookies()) == 0 {
		t.Fatal("no session cookie")
	}

	// Unknown email and wrong password share one 401 (no oracle).
	if w := login(t, api, "nobody@test.local", "password123"); w.Code != http.StatusUnauthorized {
		t.Fatalf("unknown = %d", w.Code)
	}
	if w := login(t, api, email, "wrong-password"); w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong pw = %d", w.Code)
	}
	if w := login(t, api, email, "short"); w.Code != http.StatusUnauthorized {
		t.Fatalf("bad shape = %d", w.Code)
	}
}
