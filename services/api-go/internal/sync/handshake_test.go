package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Contictus/collab-editor/services/api-go/internal/auth"
)

type stubAuthorizer struct {
	allow map[string]bool
	err   error
}

func (s stubAuthorizer) CheckDocumentAccess(_ context.Context, docID, userID string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.allow[docID+"\x00"+userID], nil
}

func signed(t *testing.T, id, email, secret string) string {
	t.Helper()
	tok, err := auth.SignSession(auth.SessionUser{ID: id, Email: email}, secret)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestParseUpgradeForms(t *testing.T) {
	// Query form.
	r := httptest.NewRequest(http.MethodGet, "/?doc=abc&token=t123", nil)
	doc, tok := ParseUpgrade(r)
	if doc != "abc" || tok != "t123" {
		t.Fatalf("query = %q %q", doc, tok)
	}
	// Path + cookie form (y-websocket browser client).
	r2 := httptest.NewRequest(http.MethodGet, "/my-doc", nil)
	r2.AddCookie(&http.Cookie{Name: auth.SessionCookie, Value: "ctok"})
	doc, tok = ParseUpgrade(r2)
	if doc != "my-doc" || tok != "ctok" {
		t.Fatalf("path = %q %q", doc, tok)
	}
	// Query doc wins over path.
	r3 := httptest.NewRequest(http.MethodGet, "/other?doc=qq", nil)
	doc, _ = ParseUpgrade(r3)
	if doc != "qq" {
		t.Fatalf("precedence = %q", doc)
	}
}

func TestAuthorizeUpgrade(t *testing.T) {
	secret := "hs-secret"
	az := stubAuthorizer{allow: map[string]bool{"d1\x00u1": true}}

	// Happy path (cookie token).
	r := httptest.NewRequest(http.MethodGet, "/d1", nil)
	r.AddCookie(&http.Cookie{Name: auth.SessionCookie, Value: signed(t, "u1", "u@x.co", secret)})
	hs, code, err := AuthorizeUpgrade(context.Background(), r, secret, az)
	if err != nil || code != 200 || hs.DocID != "d1" || hs.User.ID != "u1" {
		t.Fatalf("ok = %+v %d %v", hs, code, err)
	}

	// Missing token → 401.
	r2 := httptest.NewRequest(http.MethodGet, "/d1", nil)
	if _, code, _ := AuthorizeUpgrade(context.Background(), r2, secret, az); code != 401 {
		t.Fatalf("missing = %d", code)
	}

	// Bad token → 401.
	r3 := httptest.NewRequest(http.MethodGet, "/d1?token=bogus", nil)
	if _, code, _ := AuthorizeUpgrade(context.Background(), r3, secret, az); code != 401 {
		t.Fatalf("bogus = %d", code)
	}

	// No grant → 403 (verified identity, unshared doc).
	r4 := httptest.NewRequest(http.MethodGet, "/d2", nil)
	r4.AddCookie(&http.Cookie{Name: auth.SessionCookie, Value: signed(t, "u1", "u@x.co", secret)})
	if _, code, _ := AuthorizeUpgrade(context.Background(), r4, secret, az); code != 403 {
		t.Fatalf("forbidden = %d", code)
	}
}
