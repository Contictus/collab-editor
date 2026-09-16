package rest

import (
	"net/http"

	"github.com/Contictus/collab-editor/services/api-go/internal/auth"
)

// session resolves the request's identity (getSession parity): nil+false when
// the cookie is missing or invalid. All authenticated handlers gate on it.
func (a *API) session(r *http.Request) (*auth.SessionUser, bool) {
	user, err := auth.AuthenticateRequest(r, a.Secret)
	if err != nil {
		return nil, false
	}
	return user, true
}

// handleLogout clears the session cookie (logoutAction parity, JSON surface):
// always 204, even without a session.
func (a *API) handleLogout(w http.ResponseWriter, _ *http.Request) {
	auth.ClearSession(w, a.SecureCookies)
	w.WriteHeader(http.StatusNoContent)
}

// handleMe returns the session identity, or 401 without one.
func (a *API) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := a.session(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": user.ID, "email": user.Email})
}
