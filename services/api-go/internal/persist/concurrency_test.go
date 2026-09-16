package persist

import (
	"sync"
	"testing"
)

// TestConcurrentAppendsStayOrdered hammers Append from many goroutines: the
// worker must serialize them into gapless monotonic clocks (writeChain parity
// under connection fan-in).
func TestConcurrentAppendsStayOrdered(t *testing.T) {
	store, ctx := testStore(t)
	docID := seedDoc(t, store, ctx)
	mgr := NewManager(store, 0)
	doc, err := mgr.Load(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Destroy()
	defer mgr.Flush(docID)

	seeded := yjsUpdates(t, "z")
	const writers = 8
	const perWriter = 5
	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				mgr.Append(docID, seeded[0])
			}
		}()
	}
	wg.Wait()
	mgr.Flush(docID)

	rows, err := store.ReplayUpdates(ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != writers*perWriter {
		t.Fatalf("rows = %d", len(rows))
	}
	seen := make(map[int]bool, len(rows))
	for _, r := range rows {
		if seen[r.Clock] {
			t.Fatalf("dup clock %d", r.Clock)
		}
		seen[r.Clock] = true
	}
	for c := 0; c < writers*perWriter; c++ {
		if !seen[c] {
			t.Fatalf("gap at clock %d", c)
		}
	}
}
