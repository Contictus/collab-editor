package rest

import "net/http"

// Routes registers the JSON API on mux (Go 1.22 method+path patterns).
// Mount alongside /health and the WS handler — longest prefix wins, so
// /api/* never reaches the sync server.
func (a *API) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/register", a.handleRegister)
	mux.HandleFunc("POST /api/auth/login", a.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", a.handleLogout)
	mux.HandleFunc("GET /api/auth/me", a.handleMe)

	mux.HandleFunc("GET /api/documents", a.handleListDocuments)
	mux.HandleFunc("POST /api/documents", a.handleCreateDocument)
	mux.HandleFunc("GET /api/documents/{id}", func(w http.ResponseWriter, r *http.Request) {
		a.handleGetDocument(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("PATCH /api/documents/{id}", func(w http.ResponseWriter, r *http.Request) {
		a.handleRenameDocument(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("DELETE /api/documents/{id}", func(w http.ResponseWriter, r *http.Request) {
		a.handleDeleteDocument(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("GET /api/documents/{id}/audit", func(w http.ResponseWriter, r *http.Request) {
		a.handleAudit(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("GET /api/documents/{id}/replay", func(w http.ResponseWriter, r *http.Request) {
		a.handleReplay(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("POST /api/documents/{id}/restore", func(w http.ResponseWriter, r *http.Request) {
		a.handleRestore(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("POST /api/documents/{id}/share", func(w http.ResponseWriter, r *http.Request) {
		a.handleShareDocument(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("GET /api/documents/{id}/collaborators", func(w http.ResponseWriter, r *http.Request) {
		a.handleListCollaborators(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("DELETE /api/documents/{id}/share/{userId}", func(w http.ResponseWriter, r *http.Request) {
		a.handleUnshareDocument(w, r, r.PathValue("id"), r.PathValue("userId"))
	})
}
