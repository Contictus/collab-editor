package sync

import (
	"context"
	"testing"
)

func TestEvictHookRunsBeforeDestroy(t *testing.T) {
	reg := NewRegistry(nil)
	var hooked []*Room
	reg.SetEvictHook(func(r *Room) { hooked = append(hooked, r) })

	room, err := reg.Get(context.Background(), "d")
	if err != nil {
		t.Fatal(err)
	}
	member := &fakeSender{}
	room.Join(member)

	// Live room: no evict, no hook.
	if reg.EvictIfEmpty(room) {
		t.Fatal("evicted a live room")
	}
	if len(hooked) != 0 {
		t.Fatal("hook ran for live room")
	}

	// Empty room: hook runs exactly once, then delisted.
	if _, empty := room.Leave(member); !empty {
		t.Fatal("room should be empty")
	}
	if !reg.EvictIfEmpty(room) {
		t.Fatal("evict failed")
	}
	if len(hooked) != 1 || hooked[0] != room {
		t.Fatalf("hooked = %v", hooked)
	}
	if rooms, _ := reg.Stats(); rooms != 0 {
		t.Fatal("room still listed")
	}
}
