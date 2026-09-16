package persist

import (
	"testing"
)

func TestManagerLoadTracksCursor(t *testing.T) {
	store, ctx := testStore(t)
	docID := seedDoc(t, store, ctx)
	mgr := NewManager(store, 0)
	if mgr.threshold != DefaultThreshold {
		t.Fatalf("threshold = %d", mgr.threshold)
	}

	doc, err := mgr.Load(ctx, docID)
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}
	defer doc.Destroy()
	if got := doc.Text(); got != "" {
		t.Fatalf("text = %q", got)
	}
	p, ok := mgr.persister(docID)
	if !ok || p.clock != 0 || p.sinceSnapshot != 0 {
		t.Fatalf("cursor = %+v %v", p, ok)
	}

	// Append behind the manager's back, reload: cursor follows the log.
	seeded := yjsUpdates(t, "abc")
	for i, u := range seeded {
		if err := store.AppendUpdate(ctx, docID, u, i); err != nil {
			t.Fatal(err)
		}
	}
	doc2, err := mgr.Load(ctx, docID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer doc2.Destroy()
	if got := doc2.Text(); got != "abc" {
		t.Fatalf("text = %q", got)
	}
	p2, _ := mgr.persister(docID)
	if p2.clock != len(seeded) || p2.sinceSnapshot != len(seeded) {
		t.Fatalf("cursor = %+v", p2)
	}
}
