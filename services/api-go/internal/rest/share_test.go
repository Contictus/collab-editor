package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func shareReq(t *testing.T, api *API, id, email string, session *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/documents/"+id+"/share",
		strings.NewReader(`{"email":"`+email+`"}`))
	if session != nil {
		r.AddCookie(session)
	}
	api.handleShareDocument(w, r, id)
	return w
}

func TestShareFlow(t *testing.T) {
	api := testAPI(t)
	_, owner := authedUser(t, api, "showner")
	guestID, guest := authedUser(t, api, "shguest")
	emailOf := func(session *http.Cookie) string {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		r.AddCookie(session)
		api.handleMe(w, r)
		var me map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &me)
		return me["email"]
	}
	ownerEmail, guestEmail := emailOf(owner), emailOf(guest)
	id := createDoc(t, api, owner, "shared")

	// Guest cannot open before sharing.
	if w := getDoc(t, api, id, guest); w.Code != 404 {
		t.Fatalf("guest pre-share = %d", w.Code)
	}

	// Unknown email → 404, self-share → 409, missing doc → 404.
	if w := shareReq(t, api, id, "nobody@test.local", owner); w.Code != 404 {
		t.Fatalf("unknown email = %d", w.Code)
	}
	if w := shareReq(t, api, id, ownerEmail, owner); w.Code != 409 {
		t.Fatalf("self share = %d", w.Code)
	}
	if w := shareReq(t, api, "nope", guestEmail, owner); w.Code != 404 {
		t.Fatalf("missing doc = %d", w.Code)
	}
	// Guest sharing owner's doc → 404 (not owner).
	if w := shareReq(t, api, id, guestEmail, guest); w.Code != 404 {
		t.Fatalf("guest share = %d", w.Code)
	}

	// Share → guest opens as non-owner.
	w := shareReq(t, api, id, guestEmail, owner)
	if w.Code != http.StatusCreated {
		t.Fatalf("share = %d %s", w.Code, w.Body.String())
	}
	var grant collaboratorJSON
	if err := json.Unmarshal(w.Body.Bytes(), &grant); err != nil || grant.UserID != guestID {
		t.Fatalf("grant = %s", w.Body.String())
	}
	gw := getDoc(t, api, id, guest)
	if gw.Code != 200 {
		t.Fatalf("guest post-share = %d", gw.Code)
	}
	var detail documentDetailJSON
	_ = json.Unmarshal(gw.Body.Bytes(), &detail)
	if detail.IsOwner {
		t.Fatal("guest flagged owner")
	}

	// Collaborators list (owner-only).
	lw := httptest.NewRecorder()
	lr := httptest.NewRequest(http.MethodGet, "/api/documents/"+id+"/collaborators", nil)
	lr.AddCookie(owner)
	api.handleListCollaborators(lw, lr, id)
	var collabs []collaboratorJSON
	_ = json.Unmarshal(lw.Body.Bytes(), &collabs)
	if lw.Code != 200 || len(collabs) != 1 || collabs[0].UserID != guestID {
		t.Fatalf("collabs = %d %+v", lw.Code, collabs)
	}
	gl := httptest.NewRecorder()
	gr := httptest.NewRequest(http.MethodGet, "/api/documents/"+id+"/collaborators", nil)
	gr.AddCookie(guest)
	api.handleListCollaborators(gl, gr, id)
	if gl.Code != 404 {
		t.Fatalf("guest collabs = %d", gl.Code)
	}

	// Unshare → guest locked out again.
	uw := httptest.NewRecorder()
	ur := httptest.NewRequest(http.MethodDelete, "/api/documents/"+id+"/share/"+guestID, nil)
	ur.AddCookie(owner)
	api.handleUnshareDocument(uw, ur, id, guestID)
	if uw.Code != 204 {
		t.Fatalf("unshare = %d", uw.Code)
	}
	if w := getDoc(t, api, id, guest); w.Code != 404 {
		t.Fatalf("guest post-unshare = %d", w.Code)
	}
}
