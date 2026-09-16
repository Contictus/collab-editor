package sync

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Contictus/collab-editor/services/api-go/internal/auth"
	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
	"github.com/Contictus/collab-editor/services/api-go/internal/db"
	"github.com/Contictus/collab-editor/services/api-go/internal/persist"
)

// TestPersistedReopen is the F5 capstone: A writes through the live server,
// both clients leave (final checkpoint), a fresh registry/manager pair
// (simulated restart) serves the doc from the DB, and late joiner B
// converges through the plain sync handshake — no lost writes.
func TestPersistedReopen(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.MigrateUp(ctx, dsn); err != nil {
		t.Skipf("migrate: %v", err)
	}
	store, err := db.New(ctx, dsn)
	if err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}
	// First cleanup = runs last: pool outlives server shutdowns (evict hooks
	// finalize against it during srv.Close).
	t.Cleanup(store.Close)

	owner, err := store.CreateUser(ctx, fmt.Sprintf("go-reopen-%d@test.local", time.Now().UnixNano()), "argon2id$test")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	doc, err := store.CreateDocument(ctx, owner.ID, "reopen doc")
	if err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = store.DeleteDocument(ctx, doc.ID, owner.ID)
		_ = store.DeleteUser(ctx, owner.ID)
	})

	mkServer := func(t *testing.T) (*httptest.Server, *persist.Manager) {
		t.Helper()
		mgr := persist.NewManager(store, 100)
		reg := NewRegistry(mgr.Load)
		reg.SetEvictHook(func(r *Room) {
			_, _ = mgr.Finalize(r.ID)
		})
		srv := httptest.NewServer(NewServer(testSecret, reg, store, 0))
		t.Cleanup(srv.Close)
		return srv, mgr
	}
	token := func(id string) *http.Cookie {
		tok, err := auth.SignSession(auth.SessionUser{ID: id, Email: id + "@x.co"}, testSecret)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Cookie{Name: auth.SessionCookie, Value: tok}
	}

	// Generation 1: A writes, then everyone leaves.
	srv1, _ := mkServer(t)
	connA, _ := dial(t, wsURL(srv1, "/"+doc.ID), token(owner.ID))
	if typ, _ := readFrame(t, connA); typ != crdt.MsgSync {
		t.Fatal("no step1")
	}
	docA := crdt.New()
	defer docA.Destroy()
	var update []byte
	unsub := docA.OnUpdate(func(u []byte, _ any) { update = u })
	docA.InsertText(0, "persisted hello")
	unsub()
	if err := connA.WriteMessage(websocket.BinaryMessage, crdt.EncodeUpdateFrame(update)); err != nil {
		t.Fatal(err)
	}
	if err := connA.Close(); err != nil {
		t.Fatal(err)
	}

	// Last leave checkpoints: eventually 1 snapshot, 0 live rows.
	deadline := time.Now().Add(5 * time.Second)
	for {
		snaps, err := store.AuditSnapshots(ctx, doc.ID)
		if err != nil {
			t.Fatal(err)
		}
		live, err := store.AuditUpdates(ctx, doc.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(snaps) == 1 && len(live) == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("checkpoint = %d snaps %d live", len(snaps), len(live))
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Generation 2 (fresh managers = restart): B syncs from the DB state.
	srv2, _ := mkServer(t)
	connB, _ := dial(t, wsURL(srv2, "/"+doc.ID), token(owner.ID))
	defer connB.Close()
	if typ, _ := readFrame(t, connB); typ != crdt.MsgSync {
		t.Fatal("no step1 for B")
	}
	docB := crdt.New()
	defer docB.Destroy()
	if err := connB.WriteMessage(websocket.BinaryMessage, crdt.SyncStep1Frame(docB)); err != nil {
		t.Fatal(err)
	}
	typ, payload := readFrame(t, connB)
	if typ != crdt.MsgSync {
		t.Fatalf("B step2 type = %d", typ)
	}
	if _, err := crdt.HandleSyncMessage(docB, payload, nil); err != nil {
		t.Fatal(err)
	}
	if got := docB.Text(); got != "persisted hello" {
		t.Fatalf("B = %q", got)
	}
}
