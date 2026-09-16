package persist

import (
	"testing"
)

func TestAppendAssignsMonotonicClocks(t *testing.T) {
	store, ctx := testStore(t)
	docID := seedDoc(t, store, ctx)
	mgr := NewManager(store, 0)
	doc, err := mgr.Load(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Destroy()
	defer mgr.Flush(docID)

	seeded := yjsUpdates(t, "hello")
	for _, u := range seeded {
		mgr.Append(docID, u)
	}
	mgr.Flush(docID)

	rows, err := store.ReplayUpdates(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(seeded) {
		t.Fatalf("rows = %d, want %d", len(rows), len(seeded))
	}
	for i, r := range rows {
		if r.Clock != i {
			t.Fatalf("row %d clock = %d", i, r.Clock)
		}
	}
	p, _ := mgr.persister(docID)
	if p.clock != len(seeded) || p.sinceSnapshot != len(seeded) {
		t.Fatalf("cursor = %+v", p)
	}

	// Unknown rooms are dropped silently (evicted race).
	mgr.Append("no-such-doc", []byte{1})
}
