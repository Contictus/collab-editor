package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func createDoc(t *testing.T, api *API, session *http.Cookie, title string) string {
	t.Helper()
	w := postJSON(t, api.handleCreateDocument, "/api/documents", `{"title":"`+title+`"}`, session)
	if w.Code != http.StatusCreated {
		t.Fatalf("create = %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body["id"]
}

func TestRenameDelete(t *testing.T) {
	api := testAPI(t)
	_, owner := authedUser(t, api, "owner")
	_, guest := authedUser(t, api, "guest")
	id := createDoc(t, api, owner, "v1")

	rename := func(session *http.Cookie, title string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, "/api/documents/"+id,
			strings.NewReader(`{"title":"`+title+`"}`))
		if session != nil {
			r.AddCookie(session)
		}
		api.handleRenameDocument(w, r, id)
		return w
	}

	if w := rename(nil, "v2"); w.Code != 401 {
		t.Fatalf("anon = %d", w.Code)
	}
	if w := rename(guest, "hijack"); w.Code != 404 {
		t.Fatalf("guest = %d", w.Code)
	}
	if w := rename(owner, "v2"); w.Code != 200 {
		t.Fatalf("owner = %d %s", w.Code, w.Body.String())
	}

	del := func(session *http.Cookie) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/api/documents/"+id, nil)
		if session != nil {
			r.AddCookie(session)
		}
		api.handleDeleteDocument(w, r, id)
		return w
	}
	if w := del(guest); w.Code != 404 {
		t.Fatalf("guest del = %d", w.Code)
	}
	if w := del(owner); w.Code != 204 {
		t.Fatalf("owner del = %d", w.Code)
	}
	if w := rename(owner, "v3"); w.Code != 404 {
		t.Fatalf("renamed deleted = %d", w.Code)
	}
}
