// Command server is the Go backend entrypoint (F0 skeleton).
//
// Scope: backend Go, web (Next.js) stays. Same Postgres schema (compat).
// F0 exposes only /health; sync/auth/persistence land in F2-F6.
// Invariants #1-#5 preserved by design; see root plan.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/Contictus/collab-editor/services/api-go/internal/config"
	"github.com/Contictus/collab-editor/services/api-go/internal/db"
)

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"service": "api-go",
	})
}

func main() {
	ctx := context.Background()
	cfg, err := config.Load(false)
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrate(ctx, cfg)
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	log.Printf("[api-go] listening on :%s (GET /health)", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatal(err)
	}
}

// runMigrate applies the goose baseline then ensures FK constraints.
// `server migrate` — idempotent, safe on Prisma-managed databases.
func runMigrate(ctx context.Context, cfg config.Config) {
	mcfg, err := config.Load(true)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.MigrateUp(ctx, mcfg.DatabaseURL); err != nil {
		log.Fatalf("[api-go] migrate up: %v", err)
	}
	store, err := db.New(ctx, mcfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[api-go] migrate connect: %v", err)
	}
	defer store.Close()
	if err := store.EnsureConstraints(ctx); err != nil {
		log.Fatalf("[api-go] ensure constraints: %v", err)
	}
	log.Println("[api-go] migrate: up to date")
}
