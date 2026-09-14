import { loadText } from 'shared/crdt';

/**
 * Op-log replay planning (Faz 6). Pure — no DB — so the reconstruction logic is
 * unit-testable in isolation. Given the persisted snapshots + surviving update
 * rows, compute the state as of a target update id (event-sourcing "state at K").
 *
 * INVARIANT #3 is why replay has a floor: the op log is pruned once a covering
 * snapshot is committed, so per-update history only survives *after* the latest
 * snapshot boundary. Targets whose granular updates were pruned are not exactly
 * replayable — planReplay returns null for those rather than lying.
 */

export interface SnapshotRow {
  upToUpdateId: bigint;
  state: Uint8Array;
}

export interface UpdateRow {
  id: bigint;
  update: Uint8Array;
}

export interface ReplayPlan {
  snapshot: Uint8Array | null;
  updates: Uint8Array[];
}

/**
 * Select the snapshot + ordered updates needed to reconstruct state as of just
 * after update row `target` (target === 0n → the base snapshot state, or empty).
 * Returns null when `target` falls inside the pruned window (granular history gone).
 */
export function planReplay(
  snapshots: SnapshotRow[],
  updates: UpdateRow[],
  target: bigint,
): ReplayPlan | null {
  // Newest snapshot that does not go past the target.
  const covering = snapshots
    .filter((s) => s.upToUpdateId <= target)
    .sort((a, b) => (a.upToUpdateId < b.upToUpdateId ? -1 : 1));
  const snapshot = covering.at(-1) ?? null;
  const base = snapshot ? snapshot.upToUpdateId : -1n;

  const selected = updates
    .filter((u) => u.id > base && u.id <= target)
    .sort((a, b) => (a.id < b.id ? -1 : 1));

  // Pruned floor: a snapshot exists beyond target but none covers up to it, and no
  // surviving update reaches it → its granular state is gone. Not replayable.
  if (!snapshot && selected.length === 0 && target > 0n) {
    const prunedPast = snapshots.some((s) => s.upToUpdateId > target);
    if (prunedPast) return null;
  }

  return { snapshot: snapshot?.state ?? null, updates: selected.map((u) => u.update) };
}

/** Reconstruct the plain text as of update `target`, or null if not replayable. */
export function replayText(
  snapshots: SnapshotRow[],
  updates: UpdateRow[],
  target: bigint,
): string | null {
  const plan = planReplay(snapshots, updates, target);
  if (!plan) return null;
  return loadText(plan.snapshot, plan.updates);
}
