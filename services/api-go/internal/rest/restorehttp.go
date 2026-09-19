package rest

import (
	"net/http"
	"strconv"

	gosync "github.com/Contictus/collab-editor/services/api-go/internal/sync"
)

// handleRestore rebuilds the text as of update `at` and applies it to the
// live room (F11 history restore): 401 without session, 404 when
// inaccessible or not replayable, 400 on a bad body. Restore is an
// authorized editor op (owner OR collaborator — same gate as the history
// page and live editing), applied as live CRDT ops through RestoreText so
// broadcast + op-log append flow untouched (invariants #1, #3). History is
// never rewritten: the restored text lands as new updates.
func (a *API) handleRestore(w http.ResponseWriter, r *http.Request, id string) {
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
	if a.Rooms == nil {
		writeErr(w, http.StatusServiceUnavailable, "sync not served here")
		return
	}
	body, ok := decodeBody[map[string]string](r)
	if !ok || !isDigits(body["at"]) {
		writeErr(w, http.StatusBadRequest, "invalid `at` (expected a non-negative integer string)")
		return
	}
	target, err := strconv.ParseInt(body["at"], 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid `at` (expected a non-negative integer string)")
		return
	}
	snaps, err := a.Store.ReplaySnapshots(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	updates, err := a.Store.ReplayUpdates(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	var latest int64
	if len(updates) > 0 {
		latest = updates[len(updates)-1].ID
	} else if len(snaps) > 0 {
		latest = snaps[len(snaps)-1].UpToUpdateID
	}
	if target > latest {
		writeErr(w, http.StatusNotFound, "not replayable at that point")
		return
	}
	srows := make([]SnapshotRow, 0, len(snaps))
	for _, s := range snaps {
		srows = append(srows, SnapshotRow{UpToUpdateID: s.UpToUpdateID, State: s.State})
	}
	urows := make([]UpdateRow, 0, len(updates))
	for _, u := range updates {
		urows = append(urows, UpdateRow{ID: u.ID, Update: u.Update})
	}
	text, ok := ReplayText(srows, urows, target)
	if !ok {
		writeErr(w, http.StatusNotFound, "not replayable at that point")
		return
	}
	room, err := a.Rooms.Get(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	gosync.RestoreText(room, text)
	writeJSON(w, http.StatusOK, map[string]string{"updateId": strconv.FormatInt(target, 10), "text": text})
}
