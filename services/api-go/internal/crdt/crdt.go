// Package crdt is the Go CRDT + wire layer (F3), mirroring the Node contract:
//
//   - shared/crdt.ts: TEXT_KEY 'content', buildDoc(snapshot+updates),
//     loadText reconstruction. State is always binary updates (invariant #1).
//   - protocol MessageType: Sync=0, Awareness=1 (auth lives at the HTTP
//     upgrade, invariant #2 — never a WS message).
//   - ws-server/sync.ts framing: sync inner message appended RAW after the
//     outer type byte; awareness payload VarBytes-wrapped.
//
// The engine is reearth/ygo (pure Go, Yjs v13 V1 wire-compatible). Docs are
// authoritative server replicas (invariant #4, single instance per room in F4);
// versioning stays Yjs state vectors (invariant #5).
package crdt

import (
	"github.com/reearth/ygo/crdt"
)

// TextKey is the Y.Text key holding the document body — shared/crdt TEXT_KEY,
// also used by the CodeMirror binding on the client.
const TextKey = "content"

// Doc is an authoritative Yjs replica.
type Doc struct {
	inner *crdt.Doc
}

// New creates an empty document.
func New() *Doc {
	return &Doc{inner: crdt.New()}
}

// Destroy releases the document.
func (d *Doc) Destroy() {
	d.inner.Destroy()
}

// Apply integrates one binary V1 update (Y.applyUpdate equivalent).
func (d *Doc) Apply(update []byte) error {
	return d.inner.ApplyUpdate(update)
}

// FullState encodes the whole document (Y.encodeStateAsUpdate equivalent) —
// the snapshot bytes persisted by F5 compaction.
func (d *Doc) FullState() []byte {
	return d.inner.EncodeStateAsUpdate()
}

// StateVector encodes the document's version vector (invariant #5: Yjs owns it).
func (d *Doc) StateVector() []byte {
	return crdt.EncodeStateVectorV1(d.inner)
}

// Text returns the current plain body of the shared text.
func (d *Doc) Text() string {
	return d.inner.GetText(TextKey).ToString()
}

// InsertText appends text at index inside one transaction (edit helper).
// The text handle is obtained BEFORE Transact — GetText and Transact both
// take the document lock, so nesting them deadlocks.
func (d *Doc) InsertText(index int, s string) {
	txt := d.inner.GetText(TextKey)
	d.inner.Transact(func(txn *crdt.Transaction) {
		txt.Insert(txn, index, s, nil)
	})
}

// DeleteText removes length chars at index inside one transaction.
func (d *Doc) DeleteText(index, length int) {
	txt := d.inner.GetText(TextKey)
	d.inner.Transact(func(txn *crdt.Transaction) {
		txt.Delete(txn, index, length)
	})
}

// OnUpdate subscribes to state-mutating updates (doc.on('update')). F5 uses it
// to append to the op log and broadcast. Returns an unsubscribe func.
func (d *Doc) OnUpdate(fn func(update []byte)) func() {
	return d.inner.OnUpdate(func(update []byte, _ any) { fn(update) })
}

// BuildDoc rebuilds a document from a snapshot plus the ordered updates after
// it (load-on-open). A nil snapshot replays from the start — mirrors buildDoc.
func BuildDoc(snapshot []byte, updates [][]byte) (*Doc, error) {
	d := New()
	if snapshot != nil {
		if err := d.Apply(snapshot); err != nil {
			d.Destroy()
			return nil, err
		}
	}
	for _, u := range updates {
		if err := d.Apply(u); err != nil {
			d.Destroy()
			return nil, err
		}
	}
	return d, nil
}

// LoadText reconstructs and returns the plain text (mirrors loadText).
func LoadText(snapshot []byte, updates [][]byte) (string, error) {
	d, err := BuildDoc(snapshot, updates)
	if err != nil {
		return "", err
	}
	defer d.Destroy()
	return d.Text(), nil
}
