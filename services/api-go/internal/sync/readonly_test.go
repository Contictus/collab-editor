package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Contictus/collab-editor/services/api-go/internal/auth"
)

// roleAuthorizer extends the stub with access levels (F12 read-only gate).
type roleAuthorizer struct {
	stubAuthorizer
	roles map[string]string
}

func (s roleAuthorizer) CheckDocumentRole(_ context.Context, docID, userID string) (string, error) {
	if r, ok := s.roles[docID+"\x00"+userID]; ok {
		return r, nil
	}
	return "editor", nil
}

// Viewers handshake successfully but flagged read-only; editors and
// role-less authorizers default to full write.
func TestAuthorizeUpgradeReadOnly(t *testing.T) {
	secret := "hs-secret"
	az := roleAuthorizer{
		stubAuthorizer: stubAuthorizer{allow: map[string]bool{
			"d1\x00viewer": true,
			"d1\x00editor": true,
		}},
		roles: map[string]string{"d1\x00viewer": "viewer", "d1\x00editor": "editor"},
	}

	viewer := httptest.NewRequest(http.MethodGet, "/d1", nil)
	viewer.AddCookie(&http.Cookie{Name: auth.SessionCookie, Value: signed(t, "viewer", "v@x.co", secret)})
	hs, code, err := AuthorizeUpgrade(context.Background(), viewer, secret, az)
	if err != nil || code != 200 || !hs.ReadOnly {
		t.Fatalf("viewer = %+v %d %v", hs, code, err)
	}

	editor := httptest.NewRequest(http.MethodGet, "/d1", nil)
	editor.AddCookie(&http.Cookie{Name: auth.SessionCookie, Value: signed(t, "editor", "e@x.co", secret)})
	hs, code, err = AuthorizeUpgrade(context.Background(), editor, secret, az)
	if err != nil || code != 200 || hs.ReadOnly {
		t.Fatalf("editor = %+v %d %v", hs, code, err)
	}

	// Stubs without roles keep full write (backwards compatible).
	plain := stubAuthorizer{allow: map[string]bool{"d1\x00u1": true}}
	r := httptest.NewRequest(http.MethodGet, "/d1", nil)
	r.AddCookie(&http.Cookie{Name: auth.SessionCookie, Value: signed(t, "u1", "u@x.co", secret)})
	hs, code, err = AuthorizeUpgrade(context.Background(), r, secret, plain)
	if err != nil || code != 200 || hs.ReadOnly {
		t.Fatalf("plain = %+v %d %v", hs, code, err)
	}
}
