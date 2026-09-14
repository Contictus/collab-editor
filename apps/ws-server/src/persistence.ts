import { prisma } from 'db';
import * as Y from 'yjs';

/**
 * Document persistence (Faz 4, data-model.md). Op log is append-only; snapshots
 * are periodic checkpoints. Load rebuilds state by replaying the op log on top of
 * the latest snapshot (event-sourcing-lite).
 *
 * INVARIANT #3: a snapshot is committed BEFORE the updates it covers are pruned.
 * compact() does both inside one atomic transaction, so there is no crash window
 * where prune happened without a durable snapshot.
 */

export interface LoadResult {
  /** Next clock value to assign (max existing clock + 1). */
  clock: number;
  /** Update rows currently in the log after the latest snapshot. */
  sinceSnapshot: number;
}

/** Load-on-open: apply latest snapshot then replay newer updates into `doc`. */
export async function loadInto(doc: Y.Doc, documentId: string): Promise<LoadResult> {
  const snapshot = await prisma.documentSnapshot.findFirst({
    where: { documentId },
    orderBy: { id: 'desc' },
    select: { state: true, upToUpdateId: true },
  });
  if (snapshot) Y.applyUpdate(doc, new Uint8Array(snapshot.state));

  const updates = await prisma.documentUpdate.findMany({
    where: { documentId, ...(snapshot ? { id: { gt: snapshot.upToUpdateId } } : {}) },
    orderBy: { id: 'asc' },
    select: { update: true },
  });
  for (const u of updates) Y.applyUpdate(doc, new Uint8Array(u.update));

  const maxClock = await prisma.documentUpdate.aggregate({
    where: { documentId },
    _max: { clock: true },
  });

  return { clock: (maxClock._max.clock ?? -1) + 1, sinceSnapshot: updates.length };
}

/** Append one binary Yjs update to the op log (append-only). */
export async function appendUpdate(
  documentId: string,
  update: Uint8Array,
  clock: number,
): Promise<void> {
  await prisma.documentUpdate.create({
    data: { documentId, update: Buffer.from(update), clock },
  });
}

/**
 * Snapshot compaction: checkpoint the current authoritative state, then prune the
 * updates it covers. Atomic — snapshot insert then prune in one transaction
 * (INVARIANT #3). Returns the number of pruned update rows (0 if nothing to do).
 */
export async function compact(documentId: string, doc: Y.Doc): Promise<number> {
  const last = await prisma.documentUpdate.aggregate({
    where: { documentId },
    _max: { id: true },
  });
  const lastId = last._max.id;
  if (lastId == null) return 0; // no updates → nothing to snapshot/prune

  const state = Y.encodeStateAsUpdate(doc);

  const [, pruned] = await prisma.$transaction([
    prisma.documentSnapshot.create({
      data: { documentId, state: Buffer.from(state), upToUpdateId: lastId },
    }),
    prisma.documentUpdate.deleteMany({ where: { documentId, id: { lte: lastId } } }),
  ]);
  return pruned.count;
}
