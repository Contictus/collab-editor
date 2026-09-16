package rest

import (
	"errors"
	"net/http"

	"github.com/Contictus/collab-editor/services/api-go/internal/auth"
	"github.com/Contictus/collab-editor/services/api-go/internal/db"
)

// requireOwner enforces owner-only operations (caller-verified ownership in
// document-service): 401 without session, 404 when not owned.
func (a *API) requireOwner(w http.ResponseWriter, r *http.Request, id string) (*auth.SessionUser, bool) {
	user, ok := a.session(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return nil, false
	}
	doc, err := a.Store.GetAccessibleDocument(r.Context(), id, user.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return nil, false
	}
	if doc == nil || !doc.IsOwner {
		writeErr(w, http.StatusNotFound, "not found")
		return nil, false
	}
	return user, true
}

type shareForm struct {
	Email string `json:"email"`
}

type collaboratorJSON struct {
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt"`
}

// handleShareDocument grants a user (by email) access (shareDocument parity,
// JSON surface): 201 + grant, 404 on unknown doc/email, 409 on self-share.
func (a *API) handleShareDocument(w http.ResponseWriter, r *http.Request, id string) {
	owner, ok := a.requireOwner(w, r, id)
	if !ok {
		return
	}
	form, ok := decodeBody[shareForm](r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	invitee, err := a.Store.FindUserByEmail(r.Context(), form.Email)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if invitee == nil {
		writeErr(w, http.StatusNotFound, "No user with that email.")
		return
	}
	c, err := a.Store.ShareDocument(r.Context(), id, owner.ID, invitee.ID)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrNotOwned):
			writeErr(w, http.StatusNotFound, "Document not found.")
		case errors.Is(err, db.ErrSelfShare):
			writeErr(w, http.StatusConflict, "You already own this document.")
		default:
			writeErr(w, http.StatusInternalServerError, "internal error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, collaboratorJSON{UserID: c.UserID, Email: c.Email, CreatedAt: iso(c.CreatedAt)})
}

// handleListCollaborators returns grants, oldest first (owner-only).
func (a *API) handleListCollaborators(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := a.requireOwner(w, r, id); !ok {
		return
	}
	rows, err := a.Store.ListCollaborators(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	out := make([]collaboratorJSON, 0, len(rows))
	for _, c := range rows {
		out = append(out, collaboratorJSON{UserID: c.UserID, Email: c.Email, CreatedAt: iso(c.CreatedAt)})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleUnshareDocument revokes a grant (owner-only): always 204 on owned docs.
func (a *API) handleUnshareDocument(w http.ResponseWriter, r *http.Request, id, userID string) {
	if _, ok := a.requireOwner(w, r, id); !ok {
		return
	}
	if err := a.Store.UnshareDocument(r.Context(), id, userID); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
