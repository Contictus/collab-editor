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
	"strconv"

	"github.com/Contictus/collab-editor/services/api-go/internal/config"
	"github.com/Contictus/collab-editor/services/api-go/internal/db"
	gosync "github.com/Contictus/collab-editor/services/api-go/internal/sync"
)

func health(reg *gosync.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		rooms, conns := reg.Stats()
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"service": "api-go",
			"rooms":   rooms,
			"connections": conns,
		})
	}
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
	if len(os.Args) > 1 && os.Args[1] == "serve-ws" {
		runWS(ctx, cfg, mux)
		return
	}
	mux.HandleFunc("/health", health(gosync.NewRegistry(nil)))
	log.Printf("[api-go] listening on :%s (GET /health)", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatal(err)
	}
}

// runWS serves the y-protocols sync endpoint alongside /health.
// `server serve-ws` — the F4 live server (rooms + handshake, empty docs;
// F5 swaps the loader for op-log persistence). Dual-run safe: separate port.
func runWS(ctx context.Context, cfg config.Config, mux *http.ServeMux) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("[api-go] JWT_SECRET is not set (set it in root .env, see .env.example)")
	}
	mcfg, err := config.Load(true)
	if err != nil {
		log.Fatal(err)
	}
	store, err := db.New(ctx, mcfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[api-go] db connect: %v", err)
	}
	defer store.Close()
	var maxPayload int64
	if v := os.Getenv("WS_MAX_PAYLOAD"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxPayload = n
		}
	}
	reg := gosync.NewRegistry(nil)
	mux.HandleFunc("/health", health(reg))
	mux.Handle("/", gosync.NewServer(secret, reg, store, maxPayload))
	log.Printf("[api-go] listening on :%s (GET /health, WS y-protocols sync+awareness)", cfg.Port)
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
