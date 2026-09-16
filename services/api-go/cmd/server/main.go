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
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Contictus/collab-editor/services/api-go/internal/config"
	"github.com/Contictus/collab-editor/services/api-go/internal/db"
	"github.com/Contictus/collab-editor/services/api-go/internal/persist"
	"github.com/Contictus/collab-editor/services/api-go/internal/rest"
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
// `server serve-ws` — the F5 live server (rooms + handshake + op-log
// persistence with snapshot compaction). Dual-run safe: separate port.
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
	// No defer Close here: shutdown order is Shutdown → Close (see below).
	var maxPayload int64
	if v := os.Getenv("WS_MAX_PAYLOAD"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxPayload = n
		}
	}
	// Op-log persistence (F5): DB-backed loader + last-leave checkpoint.
	// Compaction threshold env-overridable for tests (mirrors WS_SNAPSHOT_THRESHOLD).
	threshold := persist.DefaultThreshold
	if v := os.Getenv("WS_SNAPSHOT_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			threshold = n
		}
	}
	mgr := persist.NewManager(store, threshold)
	reg := gosync.NewRegistry(mgr.Load)
	reg.SetEvictHook(func(room *gosync.Room) {
		if _, err := mgr.Finalize(room.ID); err != nil {
			log.Printf("[api-go] finalize (doc %s): %v", room.ID, err)
		}
	})
	mux.HandleFunc("/health", health(reg))
	// JSON API (F6) alongside sync: /api/* wins over the "/" WS catch-all.
	api := &rest.API{
		Store:         store,
		Secret:        secret,
		SecureCookies: os.Getenv("NODE_ENV") == "production",
		Limiter:       rest.NewRateLimiter(),
	}
	api.Routes(mux)
	syncSrv := gosync.NewServer(secret, reg, store, maxPayload)
	mux.Handle("/", syncSrv)
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: mux}
	// Graceful shutdown (F7-4): stop accepting, drop sockets (loops exit,
	// evict hooks finalize against the still-open pool), drain, close pool.
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		log.Println("[api-go] shutting down")
		shCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = srv.Shutdown(shCtx)
		syncSrv.CloseConnections()
		deadline := time.Now().Add(10 * time.Second)
		for {
			_, conns := reg.Stats()
			if conns == 0 || time.Now().After(deadline) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	log.Printf("[api-go] listening on :%s (GET /health, /api/*, WS y-protocols sync+awareness)", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	store.Close()
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
