package persist

import (
	"context"
	"sync"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
	"github.com/Contictus/collab-editor/services/api-go/internal/db"
)

// DefaultThreshold mirrors SNAPSHOT_THRESHOLD in shared (compact every N updates).
const DefaultThreshold = 100

// Persister is the per-room persistence cursor: next clock and updates since
// the last snapshot, plus the ordered worker serializing its appends.
// The ordered worker (F5-3) mutates clock/sinceSnapshot; Flush/Finalize
// (F5-7) drains through it.
type Persister struct {
	docID         string
	clock         int
	sinceSnapshot int
	worker        *Worker
}

// Manager owns per-room persisters and serves the registry Loader seam:
// Load rebuilds the authoritative doc from the op log (F5-1 state + BuildDoc).
type Manager struct {
	store     *db.Store
	threshold int

	mu         sync.Mutex
	persisters map[string]*Persister
}

// NewManager creates the manager. threshold <= 0 selects DefaultThreshold.
func NewManager(store *db.Store, threshold int) *Manager {
	if threshold <= 0 {
		threshold = DefaultThreshold
	}
	return &Manager{store: store, threshold: threshold, persisters: make(map[string]*Persister)}
}

// Load rebuilds the room doc from snapshot + replay and registers its
// persister (replacing any stale entry — a room is single-instance, so a
// second Load means the previous generation was evicted).
func (m *Manager) Load(ctx context.Context, docID string) (*crdt.Doc, error) {
	snapshot, updates, st, err := LoadState(ctx, m.store, docID)
	if err != nil {
		return nil, err
	}
	doc, err := crdt.BuildDoc(snapshot, updates)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	old := m.persisters[docID]
	m.persisters[docID] = &Persister{docID: docID, clock: st.Clock, sinceSnapshot: st.SinceSnapshot, worker: NewWorker()}
	m.mu.Unlock()
	if old != nil {
		old.worker.Stop()
	}
	return doc, nil
}

// persister returns the tracked cursor for a room.
func (m *Manager) persister(docID string) (*Persister, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.persisters[docID]
	return p, ok
}

// forget drops the cursor (after Finalize).
func (m *Manager) forget(docID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.persisters, docID)
}
