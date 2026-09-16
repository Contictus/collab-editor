package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// authedUser registers a user and returns its session cookie.
func authedUser(t *testing.T, api *API, prefix string) (string, *http.Cookie) {
	t.Helper()
	email := prefix + "-" + time.Now().Format("150405.000000000") + "@test.local"
	id, cookies := testUser(t, api, email)
	var session *http.Cookie
	for _, c := range cookies {
		if c.Name == "session" {
			session = c
		}
	}
	if session == nil {
		t.Fatal("no session cookie")
	}
	return id, session
}

func postJSON(t *testing.T, h func(http.ResponseWriter, *http.Request), path, body string, session *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	if session != nil {
		r.AddCookie(session)
	}
	h(w, r)
	return w
}

func TestListCreateDocuments(t *testing.T) {
	api := testAPI(t)
	_, session := authedUser(t, api, "docs")

	// Anonymous → 401 on both.
	if w := postJSON(t, api.handleCreateDocument, "/api/documents", `{"title":"x"}`, nil); w.Code != 401 {
		t.Fatalf("anon create = %d", w.Code)
	}
	w := httptest.NewRecorder()
	api.handleListDocuments(w, httptest.NewRequest(http.MethodGet, "/api/documents", nil))
	if w.Code != 401 {
		t.Fatalf("anon list = %d", w.Code)
	}

	// Bad titles → 400.
	for _, body := range []string{`{"title":""}`, `{"title":"` + strings.Repeat("x", 201) + `"}`, `{}`} {
		if w := postJSON(t, api.handleCreateDocument, "/api/documents", body, session); w.Code != 400 {
			t.Fatalf("body = %d", w.Code)
		}
	}

	// Create → listed as owned.
	wc := postJSON(t, api.handleCreateDocument, "/api/documents", `{"title":"My doc"}`, session)
	if wc.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", wc.Code, wc.Body.String())
	}
	var created map[string]string
	if err := json.Unmarshal(wc.Body.Bytes(), &created); err != nil || created["id"] == "" {
		t.Fatalf("created = %s", wc.Body.String())
	}
	wl := httptest.NewRecorder()
	rl := httptest.NewRequest(http.MethodGet, "/api/documents", nil)
	rl.AddCookie(session)
	api.handleListDocuments(wl, rl)
	var listed []documentJSON
	if err := json.Unmarshal(wl.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != created["id"] || !listed[0].IsOwner || listed[0].Title != "My doc" {
		t.Fatalf("listed = %+v", listed)
	}
	if _, err := time.Parse(time.RFC3339Nano, listed[0].UpdatedAt); err != nil {
		t.Fatalf("updatedAt = %q", listed[0].UpdatedAt)
	}
}
