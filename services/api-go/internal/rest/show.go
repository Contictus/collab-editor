package rest

import (
	"net/http"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
)

type documentDetailJSON struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	IsOwner   bool   `json:"isOwner"`
	Text      string `json:"text"`
	UpdatedAt string `json:"updatedAt"`
}

// handleGetDocument returns the accessible document with its current text
// (SSR bootstrap parity: latest snapshot + tail replayed, read-only).
// 401 without session, 404 when not the owner or a collaborator.
func (a *API) handleGetDocument(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := a.session(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	doc, err := a.Store.GetAccessibleDocument(r.Context(), id, user.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if doc == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	snap, err := a.Store.LatestSnapshot(r.Context(), id)
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
	updates, err := a.Store.ListUpdatesAfter(r.Context(), id, upTo)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	text, err := crdt.LoadText(snapshot, updates)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, documentDetailJSON{
		ID: doc.ID, Title: doc.Title, IsOwner: doc.IsOwner, Text: text,
		UpdatedAt: iso(doc.UpdatedAt),
	})
}
