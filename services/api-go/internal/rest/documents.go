package rest

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Contictus/collab-editor/services/api-go/internal/db"
)

// iso formats times like NextResponse.json (ISO-8601 UTC).
func iso(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

type documentJSON struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	UpdatedAt string `json:"updatedAt"`
	IsOwner   bool   `json:"isOwner"`
}

func toDocumentJSON(d db.DocumentSummary) documentJSON {
	return documentJSON{ID: d.ID, Title: d.Title, UpdatedAt: iso(d.UpdatedAt), IsOwner: d.IsOwner}
}

// handleListDocuments returns openable docs (owned + shared, newest first).
func (a *API) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	user, ok := a.session(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	docs, err := a.Store.ListDocuments(r.Context(), user.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	out := make([]documentJSON, 0, len(docs))
	for _, d := range docs {
		out = append(out, toDocumentJSON(d))
	}
	writeJSON(w, http.StatusOK, out)
}

type createDocumentForm struct {
	Title string `json:"title"`
}

// handleCreateDocument creates an owned document (createDocumentSchema parity:
// title 1..200 chars): 201 {id}, 400 on bad shape.
func (a *API) handleCreateDocument(w http.ResponseWriter, r *http.Request) {
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
	doc, err := a.Store.CreateDocument(r.Context(), user.ID, form.Title)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": doc.ID})
}
