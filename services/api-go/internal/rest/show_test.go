package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
)

func getDoc(t *testing.T, api *API, id string, session *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/documents/"+id, nil)
	if session != nil {
		r.AddCookie(session)
	}
	api.handleGetDocument(w, r, id)
	return w
}

func TestGetDocument(t *testing.T) {
	api := testAPI(t)
	_, owner := authedUser(t, api, "gowner")
	_, guest := authedUser(t, api, "gguest")
	id := createDoc(t, api, owner, "read me")

	if w := getDoc(t, api, id, nil); w.Code != 401 {
		t.Fatalf("anon = %d", w.Code)
	}
	if w := getDoc(t, api, id, guest); w.Code != 404 {
		t.Fatalf("guest = %d", w.Code)
	}

	// Seed an update behind the API, then read the rebuilt text.
	tmp := crdt.New()
	var update []byte
	unsub := tmp.OnUpdate(func(u []byte, _ any) { update = u })
	tmp.InsertText(0, "seeded text")
	unsub()
	tmp.Destroy()
	ctx := context.Background()
	if err := api.Store.AppendUpdate(ctx, id, update, 0); err != nil {
		t.Fatal(err)
	}

	w := getDoc(t, api, id, owner)
	if w.Code != 200 {
		t.Fatalf("owner = %d %s", w.Code, w.Body.String())
	}
	var detail documentDetailJSON
	if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Text != "seeded text" || !detail.IsOwner || detail.Title != "read me" {
		t.Fatalf("detail = %+v", detail)
	}
}
