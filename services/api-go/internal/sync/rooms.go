// Package sync is the Go sync server core (F4), mirroring ws-server/sync.ts:
//
//   - one authoritative in-memory Y.Doc per docId (invariant #4, single
//     instance in one process), loaded once and shared by all connections;
//   - y-protocols sync + awareness over the wire (framing in internal/crdt);
//   - awareness ephemeral: broadcast only, never persisted.
//
// F4 owns rooms + handshake + live loop. Persistence hooks (load-on-open,
// op-log append, snapshot compaction) land in F5 through the Loader seam.
package sync

import (
	"context"
	"sync"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
)

// Loader builds the authoritative doc for a room. F4's default returns an
// empty doc; F5 wires the op-log load (snapshot + replay).
type Loader func(ctx context.Context, docID string) (*crdt.Doc, error)

func emptyLoader(_ context.Context, _ string) (*crdt.Doc, error) {
	return crdt.New(), nil
}

// Sender is the minimal peer surface rooms need to broadcast. F4-3 adapts the
// WebSocket connection to it (write mutex + closed detection inside Send).
type Sender interface {
	Send(msg []byte) error
}

// Room is the authoritative live state for one document: the Y.Doc, the
// ephemeral awareness hub, and the member connections with their owned
// awareness client IDs (for disconnect cleanup).
type Room struct {
	ID       string
	doc      *crdt.Doc
	presence *crdt.Presence

	mu    sync.Mutex
	conns map[Sender]map[uint64]struct{}
}

// Document returns the authoritative replica (invariant #4: shared, not copied).
func (r *Room) Document() *crdt.Doc {
	return r.doc
}

// Presence returns the ephemeral awareness hub.
func (r *Room) Presence() *crdt.Presence {
	return r.presence
}

// Join adds a connection to the room.
func (r *Room) Join(s Sender) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.conns[s]; !ok {
		r.conns[s] = make(map[uint64]struct{})
	}
}

// NoteClient records that an awareness client ID arrived on a connection.
// Disconnect cleanup removes exactly these IDs (mirrors room.conns tracking).
func (r *Room) NoteClient(s Sender, id uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	set, ok := r.conns[s]
	if !ok {
		set = make(map[uint64]struct{})
		r.conns[s] = set
	}
	set[id] = struct{}{}
}

// Leave removes a connection, returning the awareness-removal update to
// broadcast (nil when the connection owned no states) and whether the room
// is now empty (caller evicts via the registry).
func (r *Room) Leave(s Sender) (removal []byte, empty bool) {
	r.mu.Lock()
	set := r.conns[s]
	delete(r.conns, s)
	empty = len(r.conns) == 0
	r.mu.Unlock()

	var ids []uint64
	for id := range set {
		ids = append(ids, id)
	}
	return r.presence.RemoveClients(ids), empty
}

// Members snapshots the current connections for broadcast.
func (r *Room) Members() []Sender {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Sender, 0, len(r.conns))
	for s := range r.conns {
		out = append(out, s)
	}
	return out
}

// MemberCount reports live connections (stats).
func (r *Room) MemberCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.conns)
}

// Broadcast sends msg to all members except skip. Best-effort: a failed Send
// is reported to the caller, which drops the broken connection on its loop.
func (r *Room) Broadcast(msg []byte, except Sender) {
	for _, m := range r.Members() {
		if m == except {
			continue
		}
		_ = m.Send(msg)
	}
}

// destroy releases the doc and hub. Only the registry calls this, after the
// room is unlisted and confirmed empty.
func (r *Room) destroy() {
	r.presence.Destroy()
	r.doc.Destroy()
}

// call dedupes concurrent loads of the same room (mirrors the loading map).
type call struct {
	done chan struct{}
	room *Room
	err  error
}

// Registry owns the room table. Get returns the single live instance per
// docID (invariant #4); concurrent first-opens share one load.
type Registry struct {
	mu      sync.Mutex
	rooms   map[string]*Room
	pending map[string]*call
	load    Loader
	// onEvict runs after delist, before destroy (F5 persistence finalizes
	// the checkpoint here). Unlocked — it may hit the DB.
	onEvict func(*Room)
}

// SetEvictHook registers the last-leave hook (persist.Manager.Finalize).
func (r *Registry) SetEvictHook(fn func(*Room)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onEvict = fn
}

// NewRegistry creates a registry. A nil loader serves empty docs (F4); F5
// passes the DB-backed loader.
func NewRegistry(load Loader) *Registry {
	if load == nil {
		load = emptyLoader
	}
	return &Registry{
		rooms:   make(map[string]*Room),
		pending: make(map[string]*call),
		load:    load,
	}
}

// Get returns the live room for docID, loading it once on first open.
func (r *Registry) Get(ctx context.Context, docID string) (*Room, error) {
	r.mu.Lock()
	if room, ok := r.rooms[docID]; ok {
		r.mu.Unlock()
		return room, nil
	}
	if c, ok := r.pending[docID]; ok {
		r.mu.Unlock()
		select {
		case <-c.done:
			return c.room, c.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	c := &call{done: make(chan struct{})}
	r.pending[docID] = c
	r.mu.Unlock()

	room, err := r.open(ctx, docID)
	r.mu.Lock()
	if err == nil {
		r.rooms[docID] = room
	}
	c.room, c.err = room, err
	delete(r.pending, docID)
	close(c.done)
	r.mu.Unlock()
	return room, err
}

// open builds a room outside the registry lock (load may hit the DB in F5).
// The broadcast hook attaches AFTER load returns, so replayed history is
// never rebroadcast (mirrors: load BEFORE the update listener).
func (r *Registry) open(ctx context.Context, docID string) (*Room, error) {
	doc, err := r.load(ctx, docID)
	if err != nil {
		return nil, err
	}
	room := &Room{
		ID:       docID,
		doc:      doc,
		presence: crdt.NewPresence(),
		conns:    make(map[Sender]map[uint64]struct{}),
	}
	room.doc.OnUpdate(func(u []byte, origin any) {
		var except Sender
		if s, ok := origin.(Sender); ok {
			except = s
		}
		room.Broadcast(crdt.EncodeUpdateFrame(u), except)
	})
	return room, nil
}

// EvictIfEmpty unlists and destroys the room when its last connection left
// (mirrors finalizeRoom; the evict hook checkpoints first when set).
// Returns true when the room was evicted.
func (r *Registry) EvictIfEmpty(room *Room) bool {
	r.mu.Lock()
	current, ok := r.rooms[room.ID]
	if !ok || current != room || room.MemberCount() != 0 {
		r.mu.Unlock()
		return false
	}
	delete(r.rooms, room.ID)
	hook := r.onEvict
	r.mu.Unlock()
	if hook != nil {
		hook(room)
	}
	room.destroy()
	return true
}

// Stats mirrors roomStats (health endpoint).
func (r *Registry) Stats() (rooms, conns int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, room := range r.rooms {
		rooms++
		conns += room.MemberCount()
	}
	return rooms, conns
}
