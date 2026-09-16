package persist

import (
	"context"
	"log"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
)

// maybeCompact checkpoints and prunes when the threshold is reached — mirrors
// the `sinceSnapshot >= THRESHOLD → compact` step in persistUpdate. Runs on
// the room worker (serialized with appends, like writeChain). Failures log
// and keep the log unpruned (mirrors the console.error catch); the next
// append retries.
func (m *Manager) maybeCompact(docID string, p *Persister, doc *crdt.Doc) {
	if p.sinceSnapshot < m.threshold {
		return
	}
	state := doc.FullState()
	pruned, err := m.store.Compact(context.Background(), docID, state)
	if err != nil {
		log.Printf("[persist] compact failed (doc %s): %v", docID, err)
		return
	}
	if pruned > 0 {
		p.sinceSnapshot = 0
	}
}
