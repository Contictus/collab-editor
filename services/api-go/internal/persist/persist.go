// Package persist is the Go op-log persistence layer (F5), mirroring
// ws-server/src/persistence.ts + the persistence half of sync.ts:
//
//   - load-on-open: latest snapshot + newer updates, next clock, count since
//     snapshot (LoadResult parity);
//   - append-only op log: every applied update persisted with a monotonic
//     clock, serialized through one worker (writeChain parity);
//   - snapshot compaction: checkpoint then prune in one transaction
//     (invariant #3, snapshot commit → prune).
package persist

import (
	"context"

	"github.com/Contictus/collab-editor/services/api-go/internal/db"
)

// State mirrors LoadResult in persistence.ts: the next clock to assign and
// the update rows currently in the log after the latest snapshot.
type State struct {
	// Clock is the next clock value (max existing + 1, 0 on empty log).
	Clock int
	// SinceSnapshot counts log rows after the latest snapshot.
	SinceSnapshot int
}

// LoadState reads the persisted history for a document: the latest snapshot
// blob (nil when none), the ordered updates after it, and the load state.
// Pure read — rebuilding the doc is the caller's job (Manager.Load, F5-2).
func LoadState(ctx context.Context, store *db.Store, docID string) (
	snapshot []byte, updates [][]byte, st State, err error,
) {
	snap, err := store.LatestSnapshot(ctx, docID)
	if err != nil {
		return nil, nil, State{}, err
	}
	var upTo int64
	if snap != nil {
		snapshot = snap.State
		upTo = snap.UpToUpdateID
	}
	updates, err = store.ListUpdatesAfter(ctx, docID, upTo)
	if err != nil {
		return nil, nil, State{}, err
	}
	maxClock, err := store.MaxClock(ctx, docID)
	if err != nil {
		return nil, nil, State{}, err
	}
	return snapshot, updates, State{Clock: maxClock + 1, SinceSnapshot: len(updates)}, nil
}
