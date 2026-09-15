package sync

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/Contictus/collab-editor/services/api-go/internal/auth"
)

// Handshake auth (invariant #2, mirrors ws-server/src/auth.ts + handleUpgrade).
// Runs before the socket joins any room or a single sync/awareness byte flows:
//
//	docId source: `?doc=<id>` (documented form) or the URL path segment
//	  (y-websocket puts the room name in the path);
//	token source: `?token=<jwt>` or the session cookie (same-site browser).

// Authorizer checks document access (owner OR collaborator). *db.Store
// satisfies it; tests stub it. Mirrors authorizeDocument.
type Authorizer interface {
	CheckDocumentAccess(ctx context.Context, docID, userID string) (bool, error)
}

// Handshake is an authorized upgrade request.
type Handshake struct {
	DocID string
	User  *auth.SessionUser
}

// ParseUpgrade extracts docID and token from the request (no verification).
func ParseUpgrade(r *http.Request) (docID, token string) {
	q := r.URL.Query()
	docID = q.Get("doc")
	if docID == "" {
		docID = strings.TrimLeft(r.URL.Path, "/")
		if unescaped, err := url.PathUnescape(docID); err == nil {
			docID = unescaped
		}
	}
	token = q.Get("token")
	if token == "" {
		token = auth.TokenFromRequest(r)
	}
	return docID, token
}

// AuthorizeUpgrade verifies the token then the document grant, mapping the
// outcome to the HTTP status the upgrade rejects with (401/403), like the
// Node socket destroy codes. A 500 covers authorizer (DB) failures.
func AuthorizeUpgrade(
	ctx context.Context,
	r *http.Request,
	secret string,
	az Authorizer,
) (*Handshake, int, error) {
	docID, token := ParseUpgrade(r)
	if docID == "" || token == "" {
		return nil, http.StatusUnauthorized, errMissingCredentials
	}
	user, err := auth.VerifySession(token, secret)
	if err != nil {
		return nil, http.StatusUnauthorized, err
	}
	ok, err := az.CheckDocumentAccess(ctx, docID, user.ID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if !ok {
		return nil, http.StatusForbidden, errForbidden
	}
	return &Handshake{DocID: docID, User: user}, http.StatusOK, nil
}
