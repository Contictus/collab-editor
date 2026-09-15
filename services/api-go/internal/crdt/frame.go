package crdt

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"time"

	"github.com/reearth/ygo/awareness"
	"github.com/reearth/ygo/encoding"
	"github.com/reearth/ygo/sync"
)

// Outer WS message discriminants — protocol MessageType (Sync=0, Awareness=1).
const (
	MsgSync      = 0
	MsgAwareness = 1
)

// ParseFrame splits one WS message into its outer type and inner payload.
// Sync payloads are raw inner messages; awareness payloads are VarBytes-wrapped
// (mirrors onMessage in ws-server/sync.ts).
func ParseFrame(data []byte) (msgType uint64, payload []byte, err error) {
	dec := encoding.NewDecoder(data)
	msgType, err = dec.ReadVarUint()
	if err != nil {
		return 0, nil, err
	}
	switch msgType {
	case MsgSync:
		return msgType, dec.RemainingBytesCopy(), nil
	case MsgAwareness:
		payload, err := dec.ReadVarBytes()
		if err != nil {
			return 0, nil, err
		}
		return msgType, payload, nil
	default:
		return 0, nil, errors.New("crdt: unknown message type")
	}
}

// EncodeSyncFrame wraps an inner sync message (step1/step2/update) for the wire.
func EncodeSyncFrame(inner []byte) []byte {
	return encoding.EncodeBytes(func(enc *encoding.Encoder) {
		enc.WriteVarUint(MsgSync)
		enc.WriteRaw(inner)
	})
}

// EncodeAwarenessFrame wraps an awareness update for the wire.
func EncodeAwarenessFrame(update []byte) []byte {
	return encoding.EncodeBytes(func(enc *encoding.Encoder) {
		enc.WriteVarUint(MsgAwareness)
		enc.WriteVarBytes(update)
	})
}

// EncodeUpdateFrame builds the broadcast frame for a fresh doc update
// (mirrors onDocUpdate: outer Sync + writeUpdate).
func EncodeUpdateFrame(update []byte) []byte {
	return EncodeSyncFrame(sync.EncodeUpdate(update))
}

// SyncStep1Frame is the server's handshake opener (mirrors setupConnection:
// writeSyncStep1 immediately on join).
func SyncStep1Frame(d *Doc) []byte {
	return EncodeSyncFrame(sync.EncodeSyncStep1(d.inner))
}

// HandleSyncMessage applies one inner sync message to the room doc.
// It returns the framed reply (step2 answering step1) or nil when the message
// carries no response — mirroring readSyncMessage + the length>1 send guard.
func HandleSyncMessage(d *Doc, inner []byte, origin any) (replyFrame []byte, err error) {
	reply, err := sync.ApplySyncMessage(d.inner, inner, origin)
	if err != nil {
		return nil, err
	}
	if len(reply) == 0 {
		return nil, nil
	}
	return EncodeSyncFrame(reply), nil
}

// Change mirrors the awareness change shape {added, updated, removed}.
type Change struct {
	Added   []uint64
	Updated []uint64
	Removed []uint64
}

// Presence is a room's ephemeral awareness hub. Never persisted (awareness is
// broadcast-only, like the Node room). One instance per room (F4).
type Presence struct {
	inner *awareness.Awareness
}

// NewPresence creates the hub. Like `new Awareness(doc)` on the Node side it
// owns a random client ID, but holds no local state (Node sets
// setLocalState(null)) — it only relays client presence.
func NewPresence() *Presence {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		binary.BigEndian.PutUint32(b[:], uint32(time.Now().UnixNano()))
	}
	// 32-bit like Yjs clientIDs (y-protocols varUint caps at 53 bits).
	a := awareness.New(uint64(binary.BigEndian.Uint32(b[:])))
	a.SetLocalState(nil)
	return &Presence{inner: a}
}

// Destroy releases the hub.
func (p *Presence) Destroy() {
	p.inner.Destroy()
}

// Apply integrates a client's awareness update.
func (p *Presence) Apply(update []byte, origin any) error {
	return p.inner.ApplyUpdate(update, origin)
}

// Encode returns the update for the given client IDs (nil = all states,
// used for the join-time catch-up in setupConnection).
func (p *Presence) Encode(clientIDs []uint64) []byte {
	return p.inner.EncodeUpdate(clientIDs)
}

// States returns the currently known per-client states.
func (p *Presence) States() map[uint64]awareness.ClientState {
	return p.inner.GetStates()
}

// OnChange subscribes to added/updated/removed changes (broadcast hook).
func (p *Presence) OnChange(fn func(Change, any)) func() {
	return p.inner.OnChange(func(evt awareness.ChangeEvent) {
		fn(Change{Added: evt.Added, Updated: evt.Updated, Removed: evt.Removed}, evt.Origin)
	})
}

// RemoveClients marks a disconnected connection's client IDs as removed (null
// state at their current clocks — mirrors removeAwarenessStates). It applies
// the removal locally and returns the update bytes to broadcast (nil when the
// connection owned no states).
func (p *Presence) RemoveClients(clientIDs []uint64) []byte {
	states := p.inner.GetStates()
	type doomed struct {
		id    uint64
		clock uint64
	}
	var victims []doomed
	for _, id := range clientIDs {
		if cs, ok := states[id]; ok {
			victims = append(victims, doomed{id, cs.Clock})
		}
	}
	if len(victims) == 0 {
		return nil
	}
	removal := encoding.EncodeBytes(func(enc *encoding.Encoder) {
		enc.WriteVarUint(uint64(len(victims)))
		for _, v := range victims {
			enc.WriteVarUint(v.id)
			enc.WriteVarUint(v.clock)
			enc.WriteVarString("null")
		}
	})
	if err := p.inner.ApplyUpdate(removal, nil); err != nil {
		return nil
	}
	return removal
}
