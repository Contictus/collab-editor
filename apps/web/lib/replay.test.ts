import { describe, expect, it } from 'vitest';
import * as Y from 'yjs';
import { TEXT_KEY } from 'shared/crdt';
import { type SnapshotRow, type UpdateRow, planReplay, replayText } from './replay';

/**
 * Build an op log the way the sync server does: one incremental Yjs update per edit,
 * captured off the doc's 'update' event. Returns the rows (id = 1..n) plus the
 * expected cumulative text after each step, for replay assertions.
 */
function buildOpLog(steps: string[]): { updates: UpdateRow[]; expected: string[] } {
  const doc = new Y.Doc();
  const text = doc.getText(TEXT_KEY);
  const captured: Uint8Array[] = [];
  doc.on('update', (u: Uint8Array) => captured.push(u));

  const expected: string[] = [];
  for (const s of steps) {
    text.insert(text.length, s);
    expected.push(text.toString());
  }
  doc.destroy();

  const updates = captured.map((update, i) => ({ id: BigInt(i + 1), update }));
  return { updates, expected };
}

describe('op-log replay (lib/replay)', () => {
  it('reconstructs cumulative text at each surviving update id (no snapshot)', () => {
    const { updates, expected } = buildOpLog(['Hello', ', ', 'world', '!']);

    // target 0 → empty (nothing before the first update).
    expect(replayText([], updates, 0n)).toBe('');

    for (let i = 0; i < expected.length; i++) {
      const target = BigInt(i + 1);
      expect(replayText([], updates, target)).toBe(expected[i]);
    }
  });

  it('replays across a snapshot boundary + surviving tail', () => {
    const { updates, expected } = buildOpLog(['aaa', 'bbb', 'ccc', 'ddd']);

    // Simulate compaction after update #2: snapshot = state through id 2,
    // updates 1..2 pruned, only 3..4 survive.
    const doc = new Y.Doc();
    Y.applyUpdate(doc, updates[0]!.update);
    Y.applyUpdate(doc, updates[1]!.update);
    const snapshots: SnapshotRow[] = [
      { upToUpdateId: 2n, state: Y.encodeStateAsUpdate(doc) },
    ];
    doc.destroy();
    const survivors = updates.filter((u) => u.id > 2n);

    // base (target 2) → snapshot-only state = text after step 2.
    expect(replayText(snapshots, survivors, 2n)).toBe(expected[1]);
    // target 3, 4 → snapshot + tail.
    expect(replayText(snapshots, survivors, 3n)).toBe(expected[2]);
    expect(replayText(snapshots, survivors, 4n)).toBe(expected[3]);
  });

  it('returns null for a target inside the pruned window', () => {
    const { updates } = buildOpLog(['x', 'y', 'z']);
    const doc = new Y.Doc();
    Y.applyUpdate(doc, updates[0]!.update);
    Y.applyUpdate(doc, updates[1]!.update);
    const snapshots: SnapshotRow[] = [{ upToUpdateId: 2n, state: Y.encodeStateAsUpdate(doc) }];
    doc.destroy();
    const survivors = updates.filter((u) => u.id > 2n);

    // Update #1 was pruned (covered by snapshot upTo 2) → not exactly replayable.
    expect(planReplay(snapshots, survivors, 1n)).toBeNull();
    expect(replayText(snapshots, survivors, 1n)).toBeNull();
  });
});
