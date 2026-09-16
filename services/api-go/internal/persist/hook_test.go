package persist

import (
	"testing"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
	"github.com/Contictus/collab-editor/services/api-go/internal/db"
)

// rowUpdates projects full rows to raw blobs for rebuilds.
func rowUpdates(rows []db.OpUpdate) [][]byte {
	out := make([][]byte, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Update)
	}
	return out
}

func TestLiveEditsPersist(t *testing.T) {
	store, ctx := testStore(t)
	docID := seedDoc(t, store, ctx)
	mgr := NewManager(store, 0)

	doc, err := mgr.Load(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Destroy()
	defer mgr.Flush(docID)

	// Live edit (simulates a synced client update) lands in the op log.
	doc.InsertText(0, "live")
	mgr.Flush(docID)

	rows, err := store.ReplayUpdates(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Clock != 0 {
		t.Fatalf("rows = %+v", rows)
	}
	text, err := crdt.LoadText(nil, rowUpdates(rows))
	if err != nil {
		t.Fatal(err)
	}
	if text != "live" {
		t.Fatalf("text = %q", text)
	}
}

func TestReplayNeverReappends(t *testing.T) {
	store, ctx := testStore(t)
	docID := seedDoc(t, store, ctx)
	for i, u := range yjsUpdates(t, "seed") {
		if err := store.AppendUpdate(ctx, docID, u, i); err != nil {
			t.Fatal(err)
		}
	}
	mgr := NewManager(store, 0)
	doc, err := mgr.Load(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Destroy()
	defer mgr.Flush(docID)

	// Load itself must not write: still exactly the seeded rows.
	rows, err := store.ReplayUpdates(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows after load = %d", len(rows))
	}
	// A new edit continues the clock (no overwrite).
	doc.InsertText(4, "!")
	mgr.Flush(docID)
	rows, err = store.ReplayUpdates(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[1].Clock != 1 {
		t.Fatalf("rows = %+v", rows)
	}
}
