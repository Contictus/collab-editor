package rest

import (
	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
)

// Replay planning (F6-10), mirroring replay.ts planReplay/replayText — pure,
// no DB. Given persisted snapshots + surviving update rows, compute the state
// as of just after update row target (0 → base snapshot state, or empty).
//
// Invariant #3 is why replay has a floor: pruned granular history is gone,
// and PlanReplay returns nil for it rather than lying.

// SnapshotRow mirrors SnapshotRow (ids as int64; JSON renders strings).
type SnapshotRow struct {
	UpToUpdateID int64
	State        []byte
}

// UpdateRow mirrors UpdateRow.
type UpdateRow struct {
	ID     int64
	Update []byte
}

// ReplayPlan is the snapshot + ordered updates to rebuild.
type ReplayPlan struct {
	Snapshot []byte
	Updates  [][]byte
}

// PlanReplay selects the newest snapshot not past target plus surviving
// updates in (base, target]. Nil when target falls in the pruned window.
func PlanReplay(snapshots []SnapshotRow, updates []UpdateRow, target int64) *ReplayPlan {
	var covering *SnapshotRow
	for i := range snapshots {
		if snapshots[i].UpToUpdateID <= target {
			if covering == nil || snapshots[i].UpToUpdateID > covering.UpToUpdateID {
				covering = &snapshots[i]
			}
		}
	}
	var snapshot []byte
	base := int64(-1)
	if covering != nil {
		snapshot = covering.State
		base = covering.UpToUpdateID
	}
	var selected [][]byte
	for _, u := range updates {
		if u.ID > base && u.ID <= target {
			selected = append(selected, u.Update)
		}
	}
	// Callers pass id-ordered rows (store order); selection preserves it.
	if snapshot == nil && len(selected) == 0 && target > 0 {
		for _, s := range snapshots {
			if s.UpToUpdateID > target {
				return nil
			}
		}
	}
	return &ReplayPlan{Snapshot: snapshot, Updates: selected}
}

// ReplayText reconstructs the plain text as of target, or false if not replayable.
func ReplayText(snapshots []SnapshotRow, updates []UpdateRow, target int64) (string, bool) {
	plan := PlanReplay(snapshots, updates, target)
	if plan == nil {
		return "", false
	}
	text, err := crdt.LoadText(plan.Snapshot, plan.Updates)
	if err != nil {
		return "", false
	}
	return text, true
}
