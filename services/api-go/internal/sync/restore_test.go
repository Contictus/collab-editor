package sync

import (
	"context"
	"testing"
)

// RestoreText replaces live text, including multibyte content — the delete
// length must cover the whole string or a corrupted tail survives.
func TestRestoreTextReplaces(t *testing.T) {
	reg := NewRegistry(nil)
	room, err := reg.Get(context.Background(), "restore-1")
	if err != nil {
		t.Fatal(err)
	}
	room.Document().InsertText(0, "hello-direktör ✓")
	RestoreText(room, "new text")
	if got := room.Document().Text(); got != "new text" {
		t.Fatalf("text = %q", got)
	}
}

// Restoring the same text is a no-op — no update fires, so the op log stays
// clean and subscribers see nothing.
func TestRestoreTextNoop(t *testing.T) {
	reg := NewRegistry(nil)
	room, err := reg.Get(context.Background(), "restore-2")
	if err != nil {
		t.Fatal(err)
	}
	room.Document().InsertText(0, "same")
	fired := 0
	unsub := room.Document().OnUpdate(func(_ []byte, _ any) { fired++ })
	defer unsub()
	RestoreText(room, "same")
	if fired != 0 {
		t.Fatalf("updates fired = %d", fired)
	}
}

// Restoring to empty clears the document.
func TestRestoreTextToEmpty(t *testing.T) {
	reg := NewRegistry(nil)
	room, err := reg.Get(context.Background(), "restore-3")
	if err != nil {
		t.Fatal(err)
	}
	room.Document().InsertText(0, "gone soon")
	RestoreText(room, "")
	if got := room.Document().Text(); got != "" {
		t.Fatalf("text = %q", got)
	}
}
