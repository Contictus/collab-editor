import { WebSocket as NodeWS } from 'ws';
import { WebsocketProvider } from 'y-websocket';
import * as Y from 'yjs';
import { prisma } from 'db';
import { signSession } from 'protocol';
import { TEXT_KEY } from 'shared/crdt';

/**
 * Faz 4 acceptance smoke. Requires ws-server running with WS_SNAPSHOT_THRESHOLD=5:
 *   WS_SNAPSHOT_THRESHOLD=5 pnpm --filter ws-server dev
 *   pnpm --filter ws-server exec tsx scripts/faz4-smoke.ts
 *
 * Verifies: (1) write → disconnect → reconnect reloads persisted content;
 * (2) after > N discrete updates the op log is compacted (row count drops, a
 * snapshot exists) and a fresh load still reconstructs the exact text.
 */
const WS_URL = 'ws://localhost:1234';
let failures = 0;
function check(name: string, cond: boolean, extra = '') {
  console.log(`${cond ? 'PASS' : 'FAIL'}  ${name}${extra ? '  — ' + extra : ''}`);
  if (!cond) failures++;
}
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
async function waitFor(pred: () => boolean, ms = 5000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < ms) {
    if (pred()) return true;
    await sleep(50);
  }
  return pred();
}

async function main() {
  const stamp = Date.now();
  const owner = await prisma.user.create({
    data: { email: `p4_${stamp}@ex.com`, password: 'x' },
    select: { id: true, email: true },
  });
  const doc = await prisma.document.create({
    data: { title: 'Persist Room', ownerId: owner.id },
    select: { id: true },
  });
  const token = await signSession({ id: owner.id, email: owner.email });
  const opts = {
    params: { token },
    WebSocketPolyfill: NodeWS as unknown as typeof WebSocket,
    connect: true,
  };
  const countRows = () =>
    Promise.all([
      prisma.documentUpdate.count({ where: { documentId: doc.id } }),
      prisma.documentSnapshot.count({ where: { documentId: doc.id } }),
    ]);

  // ---- Part 1: write → reload persists ----
  const docA = new Y.Doc();
  const provA = new WebsocketProvider(WS_URL, doc.id, docA, opts);
  await waitFor(() => provA.wsconnected);
  docA.getText(TEXT_KEY).insert(0, 'Persisted content');
  await sleep(600);
  provA.destroy(); // triggers final snapshot on the server
  await sleep(1000);

  const docB = new Y.Doc();
  const provB = new WebsocketProvider(WS_URL, doc.id, docB, opts);
  await waitFor(() => provB.wsconnected);
  const reloaded = await waitFor(() => docB.getText(TEXT_KEY).toString() === 'Persisted content');
  check('write → disconnect → reconnect reloads content', reloaded, JSON.stringify(docB.getText(TEXT_KEY).toString()));

  // ---- Part 2: compaction (threshold 5) ----
  const before = await countRows();
  for (let i = 0; i < 12; i++) {
    docB.getText(TEXT_KEY).insert(docB.getText(TEXT_KEY).length, `${i}`);
    await sleep(70); // discrete updates → discrete op-log rows
  }
  await sleep(1500); // let appends + compactions flush server-side

  const [updateRows, snapRows] = await countRows();
  check('snapshot(s) created by compaction', snapRows >= 1, `snapshots=${snapRows}`);
  check(
    'op log pruned (rows < 12 discrete updates)',
    updateRows < 12,
    `updateRows=${updateRows} (before part2: ${before[0]} updates / ${before[1]} snaps)`,
  );

  const expected = docB.getText(TEXT_KEY).toString();
  provB.destroy();
  await sleep(1000);

  // ---- Part 3: fresh load after compaction reconstructs exact text ----
  const docC = new Y.Doc();
  const provC = new WebsocketProvider(WS_URL, doc.id, docC, opts);
  await waitFor(() => provC.wsconnected);
  const exact = await waitFor(() => docC.getText(TEXT_KEY).toString() === expected);
  check('post-compaction load reconstructs exact text', exact, JSON.stringify(expected));
  provC.destroy();

  await sleep(300);
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
