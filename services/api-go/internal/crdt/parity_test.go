package crdt

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// yjsFixture loads testdata/yjs_parity.json — bytes produced by the real
// yjs@13.6.31 + y-protocols (see gen script in temp dir).
type yjsFixture struct {
	TextKey           string `json:"textKey"`
	FinalText         string `json:"finalText"`
	Update1           string `json:"update1"`
	Update2           string `json:"update2"`
	FullState         string `json:"fullState"`
	EmptyStateVector  string `json:"emptyStateVector"`
	Awareness         struct {
		ClientID uint64 `json:"clientID"`
		UserName string `json:"userName"`
		Update   string `json:"update"`
	} `json:"awareness"`
}

func loadYjsFixture(t *testing.T) yjsFixture {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "yjs_parity.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var f yjsFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return f
}

func b64(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("b64 decode: %v", err)
	}
	return b
}

// TestYjsUpdateParity replays real Yjs incremental updates in Go: no lost
// writes, same text. Mirrors buildDoc(nil, [u1, u2]).
func TestYjsUpdateParity(t *testing.T) {
	f := loadYjsFixture(t)
	if f.TextKey != TextKey {
		t.Fatalf("TEXT_KEY drift: fixture %q vs Go %q", f.TextKey, TextKey)
	}
	got, err := LoadText(nil, [][]byte{b64(t, f.Update1), b64(t, f.Update2)})
	if err != nil {
		t.Fatalf("LoadText: %v", err)
	}
	if got != f.FinalText {
		t.Fatalf("text = %q, want %q", got, f.FinalText)
	}
}

// TestYjsSnapshotParity applies a real Yjs full-state snapshot in Go.
func TestYjsSnapshotParity(t *testing.T) {
	f := loadYjsFixture(t)
	d, err := BuildDoc(b64(t, f.FullState), nil)
	if err != nil {
		t.Fatalf("BuildDoc: %v", err)
	}
	defer d.Destroy()
	if got := d.Text(); got != f.FinalText {
		t.Fatalf("text = %q, want %q", got, f.FinalText)
	}
	// Snapshot + already-covered update compose idempotently (replay safety).
	d2, err := BuildDoc(b64(t, f.FullState), [][]byte{b64(t, f.Update2)})
	if err != nil {
		t.Fatalf("BuildDoc snap+updates: %v", err)
	}
	defer d2.Destroy()
	if got := d2.Text(); got != f.FinalText {
		t.Fatalf("composed = %q", got)
	}
}

// TestYjsStateVectorParity checks the empty-doc state vector is byte-identical
// (both V1: a single zero varUint) — handshake roots match.
func TestYjsStateVectorParity(t *testing.T) {
	f := loadYjsFixture(t)
	d := New()
	defer d.Destroy()
	if got, want := d.StateVector(), b64(t, f.EmptyStateVector); string(got) != string(want) {
		t.Fatalf("empty SV = %x, want %x", got, want)
	}
}

// TestYjsAwarenessParity applies a real y-protocols awareness update in Go and
// reads back the client state (presence interop for F4 cursors).
func TestYjsAwarenessParity(t *testing.T) {
	f := loadYjsFixture(t)
	p := NewPresence()
	defer p.Destroy()
	typ, payload, err := ParseFrame(EncodeAwarenessFrame(b64(t, f.Awareness.Update)))
	if err != nil || typ != MsgAwareness {
		t.Fatalf("frame = %d %v", typ, err)
	}
	if err := p.Apply(payload, "js"); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	st, ok := p.States()[f.Awareness.ClientID]
	if !ok || st.State == nil {
		t.Fatalf("states = %+v", p.States())
	}
	user, _ := st.State["user"].(map[string]any)
	if user["name"] != f.Awareness.UserName {
		t.Fatalf("user = %+v", st.State["user"])
	}
}
