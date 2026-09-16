// Package rest is the Go HTTP API (F6), mirroring the Next layer:
//
//   - Server Actions (register/login/logout, document CRUD, share) become
//     JSON endpoints under /api;
//   - route handlers (audit, replay, health) keep their shapes and codes
//     (401 unauthenticated, 404 outside access, 400 bad input);
//   - BigInt ids serialize as strings, times as ISO-8601 (NextResponse.json
//     parity); errors are always {"error": "..."}.
//
// Auth is the same JWT cookie (F2); documents the same access predicate
// (owner OR collaborator, F1); text rebuilds from the op log (F5).
package rest

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Contictus/collab-editor/services/api-go/internal/db"
)

// API wires handlers to a store. SecureCookies mirrors the Node behavior
// (Secure on in production, off in dev).
type API struct {
	Store         *db.Store
	Secret        string
	SecureCookies bool
}

// maxBody caps JSON request bodies (auth/doc forms are tiny).
const maxBody = 1 << 20 // 1 MB

// writeJSON encodes v with the JSON content type and status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErr writes {"error": msg} (NextResponse.json({error}) parity).
func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// decodeBody parses a size-capped JSON request body. False on any failure
// (empty, oversized, malformed) — handlers map it to 400.
func decodeBody[T any](r *http.Request) (T, bool) {
	var v T
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	if err != nil || int64(len(body)) > maxBody {
		return v, false
	}
	if err := json.Unmarshal(body, &v); err != nil {
		return v, false
	}
	return v, true
}
