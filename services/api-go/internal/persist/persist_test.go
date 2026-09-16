package persist

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
	"github.com/Contictus/collab-editor/services/api-go/internal/db"
)

// testStore opens the shared DB or skips (same rule as db package tests).
func testStore(t *testing.T) (*db.Store, context.Context) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	if err := db.MigrateUp(ctx, dsn); err != nil {
		t.Skipf("migrate failed: %v", err)
	}
	store, err := db.New(ctx, dsn)
	if err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}
	t.Cleanup(store.Close)
	return store, ctx
}

// seedDoc creates an owner+doc and returns cleanup that deletes both.
func seedDoc(t *testing.T, store *db.Store, ctx context.Context) string {
	t.Helper()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	owner, err := store.CreateUser(ctx, "go-persist-"+suffix+"@test.local", "argon2id$test")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	doc, err := store.CreateDocument(ctx, owner.ID, "go persist doc")
	if err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = store.DeleteDocument(ctx, doc.ID, owner.ID)
		_ = store.DeleteUser(ctx, owner.ID)
	})
	return doc.ID
}

// yjsUpdates builds n real V1 updates writing text (captured via OnUpdate).
func yjsUpdates(t *testing.T, text string) [][]byte {
	t.Helper()
	d := crdt.New()
	defer d.Destroy()
	var out [][]byte
	unsub := d.OnUpdate(func(u []byte, _ any) { out = append(out, u) })
	d.InsertText(0, text)
	unsub()
	if len(out) == 0 {
		t.Fatal("no updates captured")
	}
	return out
}

func TestLoadStateEmpty(t *testing.T) {
	store, ctx := testStore(t)
	docID := seedDoc(t, store, ctx)

	snap, updates, st, err := LoadState(ctx, store, docID)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if snap != nil || len(updates) != 0 || st.Clock != 0 || st.SinceSnapshot != 0 {
		t.Fatalf("empty = %v %d %+v", snap != nil, len(updates), st)
	}
}

func TestLoadStateAfterAppends(t *testing.T) {
	store, ctx := testStore(t)
	docID := seedDoc(t, store, ctx)

	for i, u := range yjsUpdates(t, "ab") {
		if err := store.AppendUpdate(ctx, docID, u, i); err != nil {
			t.Fatalf("AppendUpdate: %v", err)
		}
	}
	snap, updates, st, err := LoadState(ctx, store, docID)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if snap != nil {
		t.Fatal("unexpected snapshot")
	}
	if st.Clock != len(updates) || st.SinceSnapshot != len(updates) || len(updates) == 0 {
		t.Fatalf("state = %+v updates = %d", st, len(updates))
	}
	// Rebuild converges to the written text (load-on-open shape).
	text, err := crdt.LoadText(snap, updates)
	if err != nil {
		t.Fatalf("LoadText: %v", err)
	}
	if text != "ab" {
		t.Fatalf("text = %q", text)
	}
}
