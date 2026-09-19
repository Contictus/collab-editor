package rest

import (
	"errors"
	"net/http"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
	"github.com/Contictus/collab-editor/services/api-go/internal/db"
)

// Public read-only links (F12). The token IS the capability: unlisted, random,
// revocable. No session — deliberately outside the auth gate (read-only text
// via op-log replay, no sync, no writes).

// handleEnablePublicLink creates (or rotates) the public token (owner-only).
func (a *API) handleEnablePublicLink(w http.ResponseWriter, r *http.Request, id string) {
	owner, ok := a.requireOwner(w, r, id)
	if !ok {
		return
	}
	publicID, err := a.Store.SetPublicID(r.Context(), id, owner.ID, true)
	if err != nil {
		if errors.Is(err, db.ErrNotOwned) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"publicId": *publicID})
}

// handleDisablePublicLink revokes the public token (owner-only).
func (a *API) handleDisablePublicLink(w http.ResponseWriter, r *http.Request, id string) {
	owner, ok := a.requireOwner(w, r, id)
	if !ok {
		return
	}
	if _, err := a.Store.SetPublicID(r.Context(), id, owner.ID, false); err != nil {
		if errors.Is(err, db.ErrNotOwned) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleGetPublicDocument returns title+text for a public token (no session).
// 404 on unknown/revoked tokens.
func (a *API) handleGetPublicDocument(w http.ResponseWriter, r *http.Request, token string) {
	doc, err := a.Store.GetByPublicID(r.Context(), token)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if doc == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	snap, err := a.Store.LatestSnapshot(r.Context(), doc.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	var snapshot []byte
	var upTo int64
	if snap != nil {
		snapshot = snap.State
		upTo = snap.UpToUpdateID
	}
	updates, err := a.Store.ListUpdatesAfter(r.Context(), doc.ID, upTo)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	text, err := crdt.LoadText(snapshot, updates)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"title": doc.Title, "text": text})
}
