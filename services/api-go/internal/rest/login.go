package rest

import (
	"net/http"

	"github.com/Contictus/collab-editor/services/api-go/internal/auth"
)

// handleLogin verifies credentials and signs in (loginAction parity, JSON
// surface): 200 + user, 401 on unknown email or wrong password (same message
// either way — no oracle), 400 on bad shape.
func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	form, ok := decodeBody[credentialsForm](r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := auth.ValidateCredentials(form.Email, form.Password); err != nil {
		writeErr(w, http.StatusUnauthorized, "Invalid email or password.")
		return
	}
	user, err := a.Store.FindUserByEmail(r.Context(), form.Email)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if user == nil || !auth.VerifyPassword(user.Password, form.Password) {
		writeErr(w, http.StatusUnauthorized, "Invalid email or password.")
		return
	}
	if err := a.issueSession(w, user); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": user.ID, "email": user.Email})
}
