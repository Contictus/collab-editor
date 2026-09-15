package sync

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
)

// Wire server (F4-3), mirrors ws-server/src/index.ts + the connection half of
// sync.ts: auth at upgrade (invariant #2), sync step1 on join, awareness
// catch-up, framed message loop, disconnect cleanup, empty-room eviction.
//
// Deliberate Node parity: no read timeout (browsers don't ping; y-websocket
// relies on TCP), 10s write deadline, maxPayload frame cap (1009 on breach).

// DefaultMaxPayload mirrors WS_MAX_PAYLOAD (1 MB).
const DefaultMaxPayload int64 = 1_000_000

// Server serves y-protocols sync + awareness on every non-/health path.
// Mount it at "/" behind /health (longest-prefix wins), like the Node http
// server that answers /health and upgrades everything else.
type Server struct {
	secret     string
	registry   *Registry
	authz      Authorizer
	maxPayload int64
	upgrader   websocket.Upgrader
}

// NewServer builds the handler. maxPayload <= 0 selects DefaultMaxPayload.
func NewServer(secret string, reg *Registry, az Authorizer, maxPayload int64) *Server {
	if maxPayload <= 0 {
		maxPayload = DefaultMaxPayload
	}
	return &Server{
		secret:     secret,
		registry:   reg,
		authz:      az,
		maxPayload: maxPayload,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			// Same-site cookie auth (like Node); no origin gate by design.
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}
}

// ServeHTTP authenticates + authorizes BEFORE upgrading (invariant #2), loads
// (or reuses) the authoritative room, then owns the connection loop.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hs, code, err := AuthorizeUpgrade(r.Context(), r, s.secret, s.authz)
	if err != nil {
		http.Error(w, http.StatusText(code), code)
		return
	}
	room, err := s.registry.Get(r.Context(), hs.DocID)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	p := &peer{conn: conn}
	room.Join(p)
	// Handshake opener (mirrors setupConnection: step1 immediately).
	if err := p.Send(crdt.SyncStep1Frame(room.Document())); err != nil {
		s.close(room, p)
		return
	}
	// Awareness catch-up when others are present.
	if states := room.Presence().States(); len(states) > 0 {
		var ids []uint64
		for id := range states {
			ids = append(ids, id)
		}
		if err := p.Send(crdt.EncodeAwarenessFrame(room.Presence().Encode(ids))); err != nil {
			s.close(room, p)
			return
		}
	}
	s.loop(room, p)
}

// loop owns the read side; Send is safe from other goroutines (write mutex).
func (s *Server) loop(room *Room, p *peer) {
	defer s.close(room, p)
	p.conn.SetReadLimit(s.maxPayload)
	for {
		msgType, data, err := p.conn.ReadMessage()
		if err != nil {
			return
		}
		if msgType != websocket.BinaryMessage {
			continue
		}
		typ, payload, err := crdt.ParseFrame(data)
		if err != nil {
			continue // unknown outer type: ignore (Node: default break)
		}
		switch typ {
		case crdt.MsgSync:
			reply, err := crdt.HandleSyncMessage(room.Document(), payload, p)
			if err != nil || reply == nil {
				continue
			}
			if err := p.Send(reply); err != nil {
				return
			}
		case crdt.MsgAwareness:
			s.onAwareness(room, p, payload)
		}
	}
}

// onAwareness applies the update, tracks ownership for disconnect cleanup,
// and broadcasts the changed states (mirrors onAwarenessUpdate).
func (s *Server) onAwareness(room *Room, p *peer, payload []byte) {
	var changed crdt.Change
	unsub := room.Presence().OnChange(func(c crdt.Change, origin any) {
		if origin == p {
			changed = c
		}
	})
	err := room.Presence().Apply(payload, p)
	unsub()
	if err != nil {
		return
	}
	for _, id := range append(append(changed.Added, changed.Updated...), changed.Removed...) {
		room.NoteClient(p, id)
	}
	ids := append(append(changed.Added, changed.Updated...), changed.Removed...)
	if len(ids) == 0 {
		return
	}
	room.Broadcast(crdt.EncodeAwarenessFrame(room.Presence().Encode(ids)), p)
}

// close removes the connection, broadcasts its awareness removal, and evicts
// the room when empty (mirrors closeConnection + finalizeRoom minus the F5
// final snapshot).
func (s *Server) close(room *Room, p *peer) {
	p.close()
	removal, empty := room.Leave(p)
	if removal != nil {
		room.Broadcast(crdt.EncodeAwarenessFrame(removal), nil)
	}
	if empty {
		s.registry.EvictIfEmpty(room)
	}
}

// peer adapts *websocket.Conn to Sender with a write mutex (concurrent
// Broadcasts from other connections' loops must not interleave frames).
type peer struct {
	conn *websocket.Conn
	wmu  sync.Mutex
	once sync.Once
}

func (p *peer) Send(msg []byte) error {
	p.wmu.Lock()
	defer p.wmu.Unlock()
	_ = p.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return p.conn.WriteMessage(websocket.BinaryMessage, msg)
}

func (p *peer) close() {
	p.once.Do(func() {
		_ = p.conn.Close()
	})
}
