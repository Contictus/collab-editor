package persist

import (
	"testing"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
)

func TestThresholdCompaction(t *testing.T) {
	store, ctx := testStore(t)
	docID := seedDoc(t, store, ctx)
	mgr := NewManager(store, 2)
	doc, err := mgr.Load(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Destroy()
	defer mgr.Flush(docID)

	// Append the same update 3 times with threshold 2: compact after the 2nd,
	// 1 live row left. (Reapplying one update is idempotent in Yjs, so the
	// text stays intact — replay safety for free.)
	seeded := yjsUpdates(t, "abc")
	for i := 0; i < 3; i++ {
		mgr.Append(docID, seeded[0])
	}
	mgr.Flush(docID)

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
	if len(live) != 1 {
		t.Fatalf("live updates = %d", len(live))
	}
	p, _ := mgr.persister(docID)
	if p.sinceSnapshot != 1 {
		t.Fatalf("sinceSnapshot = %d", p.sinceSnapshot)
	}

	// Full history still rebuilds the exact text (snapshot + tail).
	snap, updates, _, err := LoadState(ctx, store, docID)
	if err != nil {
		t.Fatal(err)
	}
	text, err := crdt.LoadText(snap, updates)
	if err != nil {
		t.Fatal(err)
	}
	if text != "abc" {
		t.Fatalf("text = %q", text)
	}
}
