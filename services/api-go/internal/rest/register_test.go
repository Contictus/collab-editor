package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Contictus/collab-editor/services/api-go/internal/db"
)

// testAPI opens the shared DB or skips (same rule as db package tests).
func testAPI(t *testing.T) *API {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	if err := db.MigrateUp(ctx, dsn); err != nil {
		t.Skipf("migrate: %v", err)
	}
	store, err := db.New(ctx, dsn)
	if err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}
	t.Cleanup(store.Close)
	return &API{Store: store, Secret: "rest-test-secret"}
}

// testUser registers a user through the API and returns its id + cookies.
func testUser(t *testing.T, api *API, email string) (string, []*http.Cookie) {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/register",
		strings.NewReader(`{"email":"`+email+`","password":"password123"}`))
	api.handleRegister(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("register %s = %d %s", email, w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		docs, _ := api.Store.ListDocuments(ctx, body["id"])
		for _, d := range docs {
			_, _ = api.Store.DeleteDocument(ctx, d.ID, body["id"])
		}
		_ = api.Store.DeleteUser(ctx, body["id"])
	})
	return body["id"], w.Result().Cookies()
}

func TestRegister(t *testing.T) {
	api := testAPI(t)
	email := "reg-" + time.Now().Format("150405.000000000") + "@test.local"
	id, cookies := testUser(t, api, email)
	if id == "" {
		t.Fatal("empty id")
	}
	var session *http.Cookie
	for _, c := range cookies {
		if c.Name == "session" {
			session = c
		}
	}
	if session == nil || session.Value == "" || !session.HttpOnly || session.Path != "/" {
		t.Fatalf("cookie = %+v", session)
	}

	// Duplicate → 409.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/register",
		strings.NewReader(`{"email":"`+email+`","password":"password123"}`))
	api.handleRegister(w, r)
	if w.Code != http.StatusConflict {
		t.Fatalf("dup = %d", w.Code)
	}

	// Bad shapes → 400.
	for _, body := range []string{`{}`, `{"email":"x","password":"short"}`, `not-json`} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
		api.handleRegister(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("body %q = %d", body, w.Code)
		}
	}
}
