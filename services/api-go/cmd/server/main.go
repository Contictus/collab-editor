// Command server is the Go backend entrypoint (F0 skeleton).
//
// Scope: backend Go, web (Next.js) stays. Same Postgres schema (compat).
// F0 exposes only /health; sync/auth/persistence land in F2-F6.
// Invariants #1-#5 preserved by design; see root plan.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"service": "api-go",
	})
}

func main() {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	log.Printf("[api-go] listening on :%s (GET /health)", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
