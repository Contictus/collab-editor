package crdt

import (
	"testing"

	"github.com/reearth/ygo/sync"
)

func TestFrameRoundTrip(t *testing.T) {
	inner := []byte{0, 1, 2, 3}
	typ, payload, err := ParseFrame(EncodeSyncFrame(inner))
	if err != nil || typ != MsgSync || string(payload) != string(inner) {
		t.Fatalf("sync frame = %d %v %v", typ, payload, err)
	}
	typ, payload, err = ParseFrame(EncodeAwarenessFrame(inner))
	if err != nil || typ != MsgAwareness || string(payload) != string(inner) {
		t.Fatalf("awareness frame = %d %v %v", typ, payload, err)
	}
	if _, _, err := ParseFrame([]byte{9}); err == nil {
		t.Fatal("unknown type accepted")
	}
	if _, _, err := ParseFrame([]byte{}); err == nil {
		t.Fatal("empty frame accepted")
	}
}

// TestHandshakeConvergence mirrors the y-protocols handshake: empty peer B
// sends step1, seeded peer A answers step2, B converges — no lost writes.
func TestHandshakeConvergence(t *testing.T) {
	a := New()
	defer a.Destroy()
	a.InsertText(0, "shared text")
	b := New()
	defer b.Destroy()

	step1 := sync.EncodeSyncStep1(b.inner)
	step2, err := sync.EncodeSyncStep2(a.inner, step1)
	if err != nil {
		t.Fatalf("step2: %v", err)
	}
	reply, err := HandleSyncMessage(b, step2, nil)
	if err != nil {
		t.Fatalf("apply step2: %v", err)
	}
	if reply != nil {
		t.Fatal("step2 should carry no reply")
	}
	if got := b.Text(); got != "shared text" {
		t.Fatalf("B = %q", got)
	}
}

// TestHandleStep1Reply checks the server side: a step1 yields a framed step2.
func TestHandleStep1Reply(t *testing.T) {
	server := New()
	defer server.Destroy()
	server.InsertText(0, "srv")
	peer := New()
	defer peer.Destroy()

	replyFrame, err := HandleSyncMessage(server, sync.EncodeSyncStep1(peer.inner), "conn-1")
	if err != nil || replyFrame == nil {
		t.Fatalf("step1 reply = %v %v", replyFrame, err)
	}
	typ, inner, err := ParseFrame(replyFrame)
	if err != nil || typ != MsgSync {
		t.Fatalf("reply frame = %d %v", typ, err)
	}
	if _, err := HandleSyncMessage(peer, inner, nil); err != nil {
		t.Fatalf("peer apply: %v", err)
	}
	if got := peer.Text(); got != "srv" {
		t.Fatalf("peer = %q", got)
	}
}

// TestUpdateBroadcastFrame checks the live-update path: an OnUpdate payload
// framed by the server applies on a peer (onDocUpdate broadcast shape).
func TestUpdateBroadcastFrame(t *testing.T) {
	server := New()
	defer server.Destroy()
	peer := New()
	defer peer.Destroy()

	// Sync first so clocks align, then broadcast the delta.
	var frames [][]byte
	unsub := server.OnUpdate(func(u []byte) { frames = append(frames, EncodeUpdateFrame(u)) })
	server.InsertText(0, "live")
	unsub()
	if len(frames) != 1 {
		t.Fatalf("broadcasts = %d", len(frames))
	}
	typ, inner, err := ParseFrame(frames[0])
	if err != nil || typ != MsgSync {
		t.Fatalf("broadcast frame = %d %v", typ, err)
	}
	if _, err := HandleSyncMessage(peer, inner, nil); err != nil {
		t.Fatalf("peer apply: %v", err)
	}
	if got := peer.Text(); got != "live" {
		t.Fatalf("peer = %q", got)
	}
}

func TestPresenceLifecycle(t *testing.T) {
	p := NewPresence()
	defer p.Destroy()
	if len(p.States()) != 0 {
		t.Fatal("fresh hub not empty")
	}

	// Simulate a client update: encode from a temp hub with local state.
	client := NewPresence()
	defer client.Destroy()
	client.inner.SetLocalState(map[string]any{"user": map[string]any{"name": "Ada"}})
	wire := EncodeAwarenessFrame(client.Encode(nil))

	typ, payload, err := ParseFrame(wire)
	if err != nil || typ != MsgAwareness {
		t.Fatalf("awareness wire = %d %v", typ, err)
	}
	var changed []Change
	unsub := p.OnChange(func(c Change, _ any) { changed = append(changed, c) })
	if err := p.Apply(payload, "conn-1"); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	unsub()
	if len(changed) == 0 || len(changed[0].Added) != 1 {
		t.Fatalf("changes = %+v", changed)
	}
	connIDs := changed[0].Added

	// Join-time catch-up encodes known states.
	if len(p.Encode(nil)) == 0 {
		t.Fatal("catch-up empty")
	}

	// Disconnect removes exactly that connection's clients.
	removal := p.RemoveClients(connIDs)
	if removal == nil {
		t.Fatal("no removal bytes")
	}
	if len(p.States()) != 0 {
		t.Fatalf("states after remove = %d", len(p.States()))
	}
	if p.RemoveClients(connIDs) != nil {
		t.Fatal("second remove should be nil")
	}
}
