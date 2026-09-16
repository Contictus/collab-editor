package rest

import (
	"net/http"
	"strings"
	"unicode/utf8"
)

// handleRenameDocument renames an owned document (renameDocument parity:
// owner-only, non-owned matches zero rows → 404).
func (a *API) handleRenameDocument(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := a.session(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	form, ok := decodeBody[createDocumentForm](r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if n := utf8.RuneCountInString(strings.TrimSpace(form.Title)); n < 1 || n > 200 {
		writeErr(w, http.StatusBadRequest, "title must be 1..200 characters")
		return
	}
	renamed, err := a.Store.RenameDocument(r.Context(), id, user.ID, form.Title)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !renamed {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": id, "title": form.Title})
}

// handleDeleteDocument deletes an owned document (deleteDocument parity:
// cascade to op log, snapshots, grants): 204, or 404 when not owned.
func (a *API) handleDeleteDocument(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := a.session(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	deleted, err := a.Store.DeleteDocument(r.Context(), id, user.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !deleted {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
