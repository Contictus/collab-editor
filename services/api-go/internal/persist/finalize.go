package persist

import (
	"context"
	"log"
)

// Finalize checkpoints a room on last leave: flush pending appends, snapshot
// the current state (pruning what it covers), stop the worker, forget the
// room — mirrors finalizeRoom. Unknown rooms are a no-op returning 0.
// Returns the pruned row count.
func (m *Manager) Finalize(docID string) (int, error) {
	p, ok := m.persister(docID)
	if !ok {
		return 0, nil
	}
	p.worker.Flush()
	m.mu.Lock()
	doc := m.docs[docID]
	m.mu.Unlock()
	pruned := 0
	if doc != nil {
		var err error
		pruned, err = m.store.Compact(context.Background(), docID, doc.FullState())
		if err != nil {
			log.Printf("[persist] final snapshot failed (doc %s): %v", docID, err)
		}
	}
	p.worker.Stop()
	m.forget(docID)
	return pruned, nil
}
