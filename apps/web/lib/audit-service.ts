import { prisma } from 'db';
import { planReplay } from './replay';
import { loadText } from 'shared/crdt';

/**
 * Op-log audit + replay (Faz 6). Reads the persisted history of a document — the
 * append-only update log and the periodic snapshots — and reconstructs the text
 * as of any surviving point (event-sourcing demo). Ownership is enforced by the
 * caller (route handler / page); these functions assume the id is authorized.
 *
 * BigInt ids are stringified for JSON safety (NextResponse.json can't serialize
 * BigInt, and they exceed Number precision in principle).
 */

export interface UpdateEntry {
  id: string;
  clock: number;
  bytes: number;
  createdAt: string;
}

export interface SnapshotEntry {
  id: string;
  upToUpdateId: string;
  bytes: number;
  createdAt: string;
}

export interface AuditLog {
  documentId: string;
  title: string;
  counts: { liveUpdates: number; snapshots: number };
  /** Snapshots oldest→newest; each is a compaction checkpoint of the op log. */
  snapshots: SnapshotEntry[];
  /** Surviving op-log rows (after the latest snapshot), oldest→newest. */
  updates: UpdateEntry[];
  /** Replay lower bound (exclusive): latest snapshot boundary, or '0' if none. */
  baseUpdateId: string;
  /** Highest surviving update id, or null if the log is empty. */
  latestUpdateId: string | null;
}

export interface ReplayResult {
  updateId: string;
  text: string;
}

/** Full audit view: counts, checkpoints, and the surviving replayable window. */
export async function getAuditLog(documentId: string, title: string): Promise<AuditLog> {
  const [snapshots, updates] = await Promise.all([
    prisma.documentSnapshot.findMany({
      where: { documentId },
      orderBy: { id: 'asc' },
      select: { id: true, upToUpdateId: true, state: true, createdAt: true },
    }),
    prisma.documentUpdate.findMany({
      where: { documentId },
      orderBy: { id: 'asc' },
      select: { id: true, clock: true, update: true, createdAt: true },
    }),
  ]);

  const base = snapshots.at(-1)?.upToUpdateId ?? 0n;
  const latest = updates.at(-1)?.id ?? null;

  return {
    documentId,
    title,
    counts: { liveUpdates: updates.length, snapshots: snapshots.length },
    snapshots: snapshots.map((s) => ({
      id: s.id.toString(),
      upToUpdateId: s.upToUpdateId.toString(),
      bytes: s.state.length,
      createdAt: s.createdAt.toISOString(),
    })),
    updates: updates.map((u) => ({
      id: u.id.toString(),
      clock: u.clock,
      bytes: u.update.length,
      createdAt: u.createdAt.toISOString(),
    })),
    baseUpdateId: base.toString(),
    latestUpdateId: latest ? latest.toString() : null,
  };
}

/**
 * Reconstruct the text as of update `updateId` (0 → base snapshot / empty).
 * Returns null when the target is out of the surviving range (pruned or beyond
 * the latest recorded update) — the endpoint maps that to 404.
 */
export async function replayTextAt(
  documentId: string,
  updateId: bigint,
): Promise<ReplayResult | null> {
  if (updateId < 0n) return null;

  const [snapshots, updates] = await Promise.all([
    prisma.documentSnapshot.findMany({
      where: { documentId },
      orderBy: { id: 'asc' },
      select: { upToUpdateId: true, state: true },
    }),
    prisma.documentUpdate.findMany({
      where: { documentId },
      orderBy: { id: 'asc' },
      select: { id: true, update: true },
    }),
  ]);

  // Beyond the latest recorded point → not a valid replay target.
  const latest = updates.at(-1)?.id ?? snapshots.at(-1)?.upToUpdateId ?? 0n;
  if (updateId > latest) return null;

  const plan = planReplay(
    snapshots.map((s) => ({ upToUpdateId: s.upToUpdateId, state: new Uint8Array(s.state) })),
    updates.map((u) => ({ id: u.id, update: new Uint8Array(u.update) })),
    updateId,
  );
  if (!plan) return null;

  return { updateId: updateId.toString(), text: loadText(plan.snapshot, plan.updates) };
}
