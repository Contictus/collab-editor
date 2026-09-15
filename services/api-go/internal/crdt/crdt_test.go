package crdt

import "testing"

func TestBuildAndLoadText(t *testing.T) {
	d := New()
	defer d.Destroy()
	d.InsertText(0, "Hello, ")
	d.InsertText(7, "world!")
	d.DeleteText(12, 1)
	if got := d.Text(); got != "Hello, world" {
		t.Fatalf("Text = %q", got)
	}

	// Full-state snapshot rebuild (compact path).
	snap := d.FullState()
	d2, err := BuildDoc(snap, nil)
	if err != nil {
		t.Fatalf("BuildDoc snapshot: %v", err)
	}
	defer d2.Destroy()
	if got := d2.Text(); got != "Hello, world" {
		t.Fatalf("rebuilt = %q", got)
	}

	// Incremental replay without snapshot.
	if got, err := LoadText(nil, [][]byte{snap}); err != nil || got != "Hello, world" {
		t.Fatalf("LoadText = %q, %v", got, err)
	}

	// Empty doc → empty text, non-nil state vector.
	e := New()
	defer e.Destroy()
	if got := e.Text(); got != "" {
		t.Fatalf("empty = %q", got)
	}
	if len(e.StateVector()) == 0 {
		t.Fatal("empty state vector")
	}
}

func TestOnUpdateFires(t *testing.T) {
	d := New()
	defer d.Destroy()
	var seen [][]byte
	unsub := d.OnUpdate(func(u []byte, _ any) { seen = append(seen, u) })
	d.InsertText(0, "x")
	unsub()
	d.InsertText(1, "y")
	if len(seen) != 1 || len(seen[0]) == 0 {
		t.Fatalf("updates seen = %d", len(seen))
	}
	// The captured update applies cleanly elsewhere.
	other := New()
	defer other.Destroy()
	if err := other.Apply(seen[0]); err != nil {
		t.Fatalf("Apply captured: %v", err)
	}
	if got := other.Text(); got != "x" {
		t.Fatalf("other = %q", got)
	}
}

func TestApplyGarbageFails(t *testing.T) {
	d := New()
	defer d.Destroy()
	if err := d.Apply([]byte{0xff, 0xff, 0xff}); err == nil {
		t.Fatal("garbage accepted")
	}
}
