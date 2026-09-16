package rest

import (
	"net/http"
	"strconv"
)

// handleReplay reconstructs the text as of update `at` (replay route parity):
// 401 without session, 404 when inaccessible, 400 on a bad `at`, 404 past the
// surviving range or inside the pruned window.
func (a *API) handleReplay(w http.ResponseWriter, r *http.Request, id string) {
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
	at := r.URL.Query().Get("at")
	if !isDigits(at) {
		writeErr(w, http.StatusBadRequest, "invalid `at` (expected a non-negative integer)")
		return
	}
	target, err := strconv.ParseInt(at, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid `at` (expected a non-negative integer)")
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
	writeJSON(w, http.StatusOK, map[string]string{"updateId": strconv.FormatInt(target, 10), "text": text})
}

// isDigits mirrors /^\d+$/ (empty is not digits).
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
