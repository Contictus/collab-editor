package rest

import (
	"net/http"
	"strconv"
)

type snapshotJSON struct {
	ID           string `json:"id"`
	UpToUpdateID string `json:"upToUpdateId"`
	Bytes        int    `json:"bytes"`
	CreatedAt    string `json:"createdAt"`
}

type updateJSON struct {
	ID        string `json:"id"`
	Clock     int    `json:"clock"`
	Bytes     int    `json:"bytes"`
	CreatedAt string `json:"createdAt"`
}

type auditJSON struct {
	DocumentID string `json:"documentId"`
	Title      string `json:"title"`
	Counts     struct {
		LiveUpdates int `json:"liveUpdates"`
		Snapshots   int `json:"snapshots"`
	} `json:"counts"`
	Snapshots      []snapshotJSON `json:"snapshots"`
	Updates        []updateJSON   `json:"updates"`
	BaseUpdateID   string         `json:"baseUpdateId"`
	LatestUpdateID *string        `json:"latestUpdateId"`
}

// handleAudit returns the persisted history (getAuditLog parity): counts,
// checkpoints oldest→newest, surviving rows oldest→newest, replay window.
// 401 without session, 404 when inaccessible.
func (a *API) handleAudit(w http.ResponseWriter, r *http.Request, id string) {
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
	out := auditJSON{DocumentID: id, Title: doc.Title, BaseUpdateID: "0"}
	out.Snapshots = make([]snapshotJSON, 0, len(snaps))
	for _, s := range snaps {
		out.Snapshots = append(out.Snapshots, snapshotJSON{
			ID: strconv.FormatInt(s.ID, 10),
			UpToUpdateID: strconv.FormatInt(s.UpToUpdateID, 10),
			Bytes: len(s.State), CreatedAt: iso(s.CreatedAt),
		})
		out.BaseUpdateID = strconv.FormatInt(s.UpToUpdateID, 10)
	}
	out.Updates = make([]updateJSON, 0, len(updates))
	for _, u := range updates {
		out.Updates = append(out.Updates, updateJSON{
			ID: strconv.FormatInt(u.ID, 10), Clock: u.Clock,
			Bytes: len(u.Update), CreatedAt: iso(u.CreatedAt),
		})
		out.LatestUpdateID = ptr(strconv.FormatInt(u.ID, 10))
	}
	out.Counts.LiveUpdates = len(updates)
	out.Counts.Snapshots = len(snaps)
	writeJSON(w, http.StatusOK, out)
}

func ptr[T any](v T) *T { return &v }
