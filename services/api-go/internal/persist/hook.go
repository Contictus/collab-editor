package persist

import (
	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
)

// Attach subscribes a room doc: every state-mutating update is queued to the
// op log (mirrors doc.on('update') → persistUpdate in sync.ts). Broadcast is
// a separate subscriber owned by the registry — the two never interleave.
// Returns the unsubscribe func (room eviction calls it before destroy).
func (m *Manager) Attach(docID string, doc *crdt.Doc) func() {
	return doc.OnUpdate(func(u []byte, _ any) { m.Append(docID, u) })
}
