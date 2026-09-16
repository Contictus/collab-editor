import * as Y from 'yjs';
import { prisma } from 'db';
import { TEXT_KEY } from 'shared/crdt';
import { getAuditLog, replayTextAt } from '../lib/audit-service';

/**
 * Faz 6 acceptance smoke (real DB, no running server). Seeds an op log of known
 * incremental updates, then verifies:
 *   (1) getAuditLog reports the right counts + surviving window;
 *   (2) replayTextAt reconstructs the exact text at each known point;
 *   (3) after a snapshot+prune, replay works across the boundary and returns
 *       null inside the pruned window (honest event-sourcing floor).
 *
 * Run:  pnpm --filter web exec tsx scripts/faz6-smoke.ts
 */
let failures = 0;
function check(name: string, cond: boolean, extra = '') {
  console.log(`${cond ? 'PASS' : 'FAIL'}  ${name}${extra ? '  — ' + extra : ''}`);
  if (!cond) failures++;
}

/** One incremental Yjs update per step (mirrors op-log appendUpdate rows). */
function buildOpLog(steps: string[]): { updates: Uint8Array[]; expected: string[] } {
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
  return { updates: captured, expected };
}

async function main() {
  const stamp = Date.now();
  const owner = await prisma.user.create({
    data: { email: `p6_${stamp}@ex.com`, password: 'x' },
    select: { id: true, email: true },
  });
  const doc = await prisma.document.create({
    data: { title: 'Replay Room', ownerId: owner.id },
    select: { id: true, title: true },
  });

  const steps = ['# Title\n', 'first line. ', 'second line. ', 'third line.'];
  const { updates, expected } = buildOpLog(steps);

  // Append rows in order (ids autoincrement asc → clock = i).
  const ids: bigint[] = [];
  for (let i = 0; i < updates.length; i++) {
    const row = await prisma.documentUpdate.create({
      data: { documentId: doc.id, update: Buffer.from(updates[i]!), clock: i },
      select: { id: true },
    });
    ids.push(row.id);
  }

  // ---- (1) audit log ----
  const audit = await getAuditLog(doc.id, doc.title);
  check('audit counts live updates', audit.counts.liveUpdates === steps.length, `liveUpdates=${audit.counts.liveUpdates}`);
  check('audit has no snapshots yet', audit.counts.snapshots === 0);
  check('audit baseUpdateId is 0', audit.baseUpdateId === '0', `base=${audit.baseUpdateId}`);
  check('audit latestUpdateId is last row', audit.latestUpdateId === ids.at(-1)!.toString());

  // ---- (2) replay at each known point ----
  const empty = await replayTextAt(doc.id, 0n);
  check('replay at 0 → empty', empty?.text === '', JSON.stringify(empty?.text));
  for (let i = 0; i < ids.length; i++) {
    const r = await replayTextAt(doc.id, ids[i]!);
    check(`replay at update #${ids[i]} → step ${i} text`, r?.text === expected[i], JSON.stringify(r?.text));
  }
  const beyond = await replayTextAt(doc.id, ids.at(-1)! + 100n);
  check('replay beyond latest → null', beyond === null);

  // ---- (3) snapshot + prune, then replay across the boundary ----
  const cutoff = ids[1]!; // snapshot covers updates through the 2nd row
  const snapDoc = new Y.Doc();
  Y.applyUpdate(snapDoc, updates[0]!);
  Y.applyUpdate(snapDoc, updates[1]!);
  await prisma.$transaction([
    prisma.documentSnapshot.create({
      data: { documentId: doc.id, state: Buffer.from(Y.encodeStateAsUpdate(snapDoc)), upToUpdateId: cutoff },
    }),
    prisma.documentUpdate.deleteMany({ where: { documentId: doc.id, id: { lte: cutoff } } }),
  ]);
  snapDoc.destroy();

  const audit2 = await getAuditLog(doc.id, doc.title);
  check('post-compaction: 1 snapshot', audit2.counts.snapshots === 1);
  check('post-compaction: base moved to cutoff', audit2.baseUpdateId === cutoff.toString(), `base=${audit2.baseUpdateId}`);
  check('post-compaction: only tail survives', audit2.counts.liveUpdates === ids.length - 2, `live=${audit2.counts.liveUpdates}`);

  const atBase = await replayTextAt(doc.id, cutoff);
  check('replay at snapshot base → step 1 text', atBase?.text === expected[1], JSON.stringify(atBase?.text));
  const atTail = await replayTextAt(doc.id, ids.at(-1)!);
  check('replay at tail (snapshot + tail) → final text', atTail?.text === expected.at(-1), JSON.stringify(atTail?.text));
  const pruned = await replayTextAt(doc.id, ids[0]!);
  check('replay inside pruned window → null', pruned === null);

  // cleanup
  await prisma.documentUpdate.deleteMany({ where: { documentId: doc.id } });
  await prisma.documentSnapshot.deleteMany({ where: { documentId: doc.id } });
  await prisma.document.delete({ where: { id: doc.id } });
  await prisma.user.delete({ where: { id: owner.id } });
  await prisma.$disconnect();

  console.log(failures === 0 ? '\nALL PASS' : `\n${failures} FAILED`);
  process.exit(failures === 0 ? 0 : 1);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
