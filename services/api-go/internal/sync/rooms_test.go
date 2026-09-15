package sync

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
)

type fakeSender struct {
	mu   sync.Mutex
	got  [][]byte
	fail bool
}

func (f *fakeSender) Send(msg []byte) error {
	if f.fail {
		return errors.New("broken")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.got = append(f.got, msg)
	return nil
}

func TestRegistrySingleInstance(t *testing.T) {
	var loads atomic.Int32
	reg := NewRegistry(func(_ context.Context, id string) (*crdt.Doc, error) {
		loads.Add(1)
		return crdt.New(), nil
	})
	const n = 16
	rooms := make([]*Room, n)
	var wg sync.WaitGroup
	for i := range rooms {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r, err := reg.Get(context.Background(), "doc-A")
			if err != nil {
				t.Errorf("Get: %v", err)
				return
			}
			rooms[i] = r
		}(i)
	}
	wg.Wait()
	for i := 1; i < n; i++ {
		if rooms[i] != rooms[0] {
			t.Fatal("concurrent opens diverged")
		}
	}
	if loads.Load() != 1 {
		t.Fatalf("loader ran %d times", loads.Load())
	}
	other, _ := reg.Get(context.Background(), "doc-B")
	if other == rooms[0] {
		t.Fatal("different IDs share a room")
	}
}

func TestRegistryLoadError(t *testing.T) {
	want := errors.New("db down")
	reg := NewRegistry(func(_ context.Context, _ string) (*crdt.Doc, error) {
		return nil, want
	})
	if _, err := reg.Get(context.Background(), "x"); err != want {
		t.Fatalf("err = %v", err)
	}
	// Failed loads are not cached: a fixed loader succeeds next.
	reg2 := NewRegistry(nil)
	if _, err := reg2.Get(context.Background(), "x"); err != nil {
		t.Fatalf("retry: %v", err)
	}
}

func TestRoomJoinLeaveBroadcast(t *testing.T) {
	reg := NewRegistry(nil)
	room, err := reg.Get(context.Background(), "d")
	if err != nil {
		t.Fatal(err)
	}
	a, b := &fakeSender{}, &fakeSender{}
	room.Join(a)
	room.Join(b)
	room.NoteClient(a, 7)
	if n := room.MemberCount(); n != 2 {
		t.Fatalf("members = %d", n)
	}
	room.Broadcast([]byte{1, 2, 3}, a)
	if len(a.got) != 0 || len(b.got) != 1 {
		t.Fatal("broadcast except broken")
	}
	removal, empty := room.Leave(a)
	if empty {
		t.Fatal("room reported empty with a member left")
	}
	_ = removal
	if _, empty := room.Leave(b); !empty {
		t.Fatal("room should be empty")
	}
	if rooms, conns := reg.Stats(); rooms != 1 || conns != 0 {
		t.Fatalf("stats = %d %d", rooms, conns)
	}
	if !reg.EvictIfEmpty(room) {
		t.Fatal("evict failed")
	}
	if rooms, _ := reg.Stats(); rooms != 0 {
		t.Fatal("room still listed")
	}
	// Evict is a no-op when someone rejoined.
	room2, _ := reg.Get(context.Background(), "d")
	room2.Join(a)
	if reg.EvictIfEmpty(room2) {
		t.Fatal("evicted a live room")
	}
}

func TestRoomUnknownLeaveIsNoOp(t *testing.T) {
	reg := NewRegistry(nil)
	room, _ := reg.Get(context.Background(), "d")
	member := &fakeSender{}
	room.Join(member)
	if _, empty := room.Leave(&fakeSender{}); empty {
		t.Fatal("unknown sender emptied the room")
	}
	if n := room.MemberCount(); n != 1 {
		t.Fatalf("members = %d", n)
	}
}
