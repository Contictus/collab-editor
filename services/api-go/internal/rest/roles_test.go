package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
)

func shareRoleReq(t *testing.T, api *API, id, email, role string, session *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/documents/"+id+"/share",
		strings.NewReader(`{"email":"`+email+`","role":"`+role+`"}`))
	if session != nil {
		r.AddCookie(session)
	}
	api.handleShareDocument(w, r, id)
	return w
}

// Sharing with a viewer role stores the role; unknown roles are 400.
func TestShareRole(t *testing.T) {
	api := testAPI(t)
	_, owner := authedUser(t, api, "rlowner")
	guestID, guest := authedUser(t, api, "rlguest")
	_ = guestID
	emailOf := func(session *http.Cookie) string {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		r.AddCookie(session)
		api.handleMe(w, r)
		var me map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &me)
		return me["email"]
	}
	guestEmail := emailOf(guest)
	id := createDoc(t, api, owner, "roles")

	if w := shareRoleReq(t, api, id, guestEmail, "admin", owner); w.Code != 400 {
		t.Fatalf("bad role = %d", w.Code)
	}
	w := shareRoleReq(t, api, id, guestEmail, "viewer", owner)
	var c map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &c)
	if w.Code != 201 || c["role"] != "viewer" {
		t.Fatalf("viewer share = %d %s", w.Code, w.Body.String())
	}
	// Role upgrade to editor on re-share.
	w = shareRoleReq(t, api, id, guestEmail, "editor", owner)
	_ = json.Unmarshal(w.Body.Bytes(), &c)
	if w.Code != 201 || c["role"] != "editor" {
		t.Fatalf("role upgrade = %d %s", w.Code, w.Body.String())
	}
	// Store-level role check.
	role, err := api.Store.CheckDocumentRole(context.Background(), id, guestID)
	if err != nil || role != "editor" {
		t.Fatalf("role = %q %v", role, err)
	}
}

// Public links: enable → read without auth → disable → gone.
func TestPublicLinkFlow(t *testing.T) {
	api := testAPI(t)
	_, owner := authedUser(t, api, "pubowner")
	id := createDoc(t, api, owner, "public doc")

	// Seed one update so the public read has text.
	tmp := crdt.New()
	var updates [][]byte
	unsub := tmp.OnUpdate(func(u []byte, _ any) { updates = append(updates, u) })
	tmp.InsertText(0, "hello public")
	unsub()
	tmp.Destroy()
	ctx := context.Background()
	for i, u := range updates {
		if err := api.Store.AppendUpdate(ctx, id, u, i); err != nil {
			t.Fatal(err)
		}
	}

	// Enable (owner).
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/documents/"+id+"/public", nil)
	r.AddCookie(owner)
	api.handleEnablePublicLink(w, r, id)
	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if w.Code != 201 || body["publicId"] == "" {
		t.Fatalf("enable = %d %s", w.Code, w.Body.String())
	}
	token := body["publicId"]

	// Read without any session.
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/api/public/"+token, nil)
	api.handleGetPublicDocument(w2, r2, token)
	var pub map[string]string
	_ = json.Unmarshal(w2.Body.Bytes(), &pub)
	if w2.Code != 200 || pub["title"] != "public doc" || pub["text"] != "hello public" {
		t.Fatalf("public read = %d %s", w2.Code, w2.Body.String())
	}

	// Unknown token → 404.
	w3 := httptest.NewRecorder()
	r3 := httptest.NewRequest(http.MethodGet, "/api/public/nope", nil)
	api.handleGetPublicDocument(w3, r3, "nope")
	if w3.Code != 404 {
		t.Fatalf("unknown token = %d", w3.Code)
	}

	// Disable → token dead.
	w4 := httptest.NewRecorder()
	r4 := httptest.NewRequest(http.MethodDelete, "/api/documents/"+id+"/public", nil)
	r4.AddCookie(owner)
	api.handleDisablePublicLink(w4, r4, id)
	if w4.Code != 204 {
		t.Fatalf("disable = %d", w4.Code)
	}
	w5 := httptest.NewRecorder()
	r5 := httptest.NewRequest(http.MethodGet, "/api/public/"+token, nil)
	api.handleGetPublicDocument(w5, r5, token)
	if w5.Code != 404 {
		t.Fatalf("revoked read = %d", w5.Code)
	}
}
