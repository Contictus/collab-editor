package persist

import (
	"testing"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
)

func TestFinalizeCheckpoints(t *testing.T) {
	store, ctx := testStore(t)
	docID := seedDoc(t, store, ctx)
	mgr := NewManager(store, 0)

	doc, err := mgr.Load(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	doc.InsertText(0, "final")
	pruned, err := mgr.Finalize(docID)
	if err != nil {
		t.Fatal(err)
	}
	doc.Destroy()
	if pruned != 1 {
		t.Fatalf("pruned = %d", pruned)
	}

	// Checkpoint durable: snapshot + empty tail, text intact.
	snaps, err := store.AuditSnapshots(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 1 {
		t.Fatalf("snapshots = %d", len(snaps))
	}
	live, err := store.AuditUpdates(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	if len(live) != 0 {
		t.Fatalf("live = %d", len(live))
	}
	snap, updates, _, err := LoadState(ctx, store, docID)
	if err != nil {
		t.Fatal(err)
	}
	text, err := crdt.LoadText(snap, updates)
	if err != nil {
		t.Fatal(err)
	}
	if text != "final" {
		t.Fatalf("text = %q", text)
	}

	// Second finalize is a no-op (room forgotten).
	if pruned, err := mgr.Finalize(docID); err != nil || pruned != 0 {
		t.Fatalf("repeat = %d %v", pruned, err)
	}
	if _, ok := mgr.persister(docID); ok {
		t.Fatal("persister not forgotten")
	}
}
