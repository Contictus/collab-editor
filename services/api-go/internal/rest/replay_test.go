package rest

import (
	"testing"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
)

// scenario builds two sequential updates ("Hello", " world") plus the full
// state after the first (a snapshot covering update 1).
func replayScenario(t *testing.T) (u1, u2, snap1 []byte) {
	t.Helper()
	d := crdt.New()
	defer d.Destroy()
	var updates [][]byte
	unsub := d.OnUpdate(func(u []byte, _ any) { updates = append(updates, u) })
	d.InsertText(0, "Hello")
	snap1 = d.FullState()
	d.InsertText(5, " world")
	unsub()
	if len(updates) != 2 {
		t.Fatalf("updates = %d", len(updates))
	}
	return updates[0], updates[1], snap1
}

func TestPlanReplayNoSnapshot(t *testing.T) {
	u1, u2, _ := replayScenario(t)
	updates := []UpdateRow{{ID: 1, Update: u1}, {ID: 2, Update: u2}}

	if text, ok := ReplayText(nil, updates, 1); !ok || text != "Hello" {
		t.Fatalf("at 1 = %q %v", text, ok)
	}
	if text, ok := ReplayText(nil, updates, 2); !ok || text != "Hello world" {
		t.Fatalf("at 2 = %q %v", text, ok)
	}
	if text, ok := ReplayText(nil, updates, 0); !ok || text != "" {
		t.Fatalf("at 0 = %q %v", text, ok)
	}
}

func TestPlanReplayWithSnapshot(t *testing.T) {
	u1, u2, snap1 := replayScenario(t)
	snaps := []SnapshotRow{{UpToUpdateID: 1, State: snap1}}
	updates := []UpdateRow{{ID: 2, Update: u2}}

	if text, ok := ReplayText(snaps, updates, 1); !ok || text != "Hello" {
		t.Fatalf("at 1 = %q %v", text, ok)
	}
	if text, ok := ReplayText(snaps, updates, 2); !ok || text != "Hello world" {
		t.Fatalf("at 2 = %q %v", text, ok)
	}
	_ = u1
}

func TestPlanReplayPrunedFloor(t *testing.T) {
	_, u2, snap1 := replayScenario(t)
	// Snapshot covers past update 2's range but no snapshot reaches target 1
	// and no surviving row does either → not replayable.
	snaps := []SnapshotRow{{UpToUpdateID: 5, State: snap1}}
	updates := []UpdateRow{{ID: 6, Update: u2}}
	if _, ok := ReplayText(snaps, updates, 1); ok {
		t.Fatal("pruned target replayable")
	}
}
