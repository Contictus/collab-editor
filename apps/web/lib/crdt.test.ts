import { describe, expect, it } from 'vitest';
import * as Y from 'yjs';
import { TEXT_KEY, loadText } from 'shared/crdt';

function updateFrom(mutate: (t: Y.Text) => void): Uint8Array {
  const doc = new Y.Doc();
  mutate(doc.getText(TEXT_KEY));
  return Y.encodeStateAsUpdate(doc);
}

describe('CRDT reconstruction (shared/crdt)', () => {
  it('returns empty string for no snapshot and no updates', () => {
    expect(loadText(null, [])).toBe('');
  });

  it('replays a single update', () => {
    const u = updateFrom((t) => t.insert(0, 'Hello, world'));
    expect(loadText(null, [u])).toBe('Hello, world');
  });

  it('merges snapshot + later updates', () => {
    const snapshot = updateFrom((t) => t.insert(0, 'base '));
    // A later update authored against the same content, appended.
    const doc = new Y.Doc();
    Y.applyUpdate(doc, snapshot);
    doc.getText(TEXT_KEY).insert(5, 'more');
    const laterUpdate = Y.encodeStateAsUpdate(doc);
    expect(loadText(snapshot, [laterUpdate])).toBe('base more');
  });
});
