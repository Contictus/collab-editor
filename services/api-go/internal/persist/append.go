package persist

import (
	"context"
)

// Append queues one binary update for the op log (fire-and-forget, ordered).
// The worker assigns the monotonic clock and bumps sinceSnapshot — mirroring
// persistUpdate: serialized appends keep clock monotonic. Unknown docIDs are
// dropped (room already evicted).
func (m *Manager) Append(docID string, update []byte) {
	p, ok := m.persister(docID)
	if !ok {
		return
	}
	owned := append([]byte(nil), update...)
	p.worker.Go(func() {
		ctx := context.Background()
		_ = m.store.AppendUpdate(ctx, docID, owned, p.clock)
		p.clock++
		p.sinceSnapshot++
	})
}

// Flush waits until the room's queued appends complete (tests + Finalize).
func (m *Manager) Flush(docID string) {
	if p, ok := m.persister(docID); ok {
		p.worker.Flush()
	}
}
