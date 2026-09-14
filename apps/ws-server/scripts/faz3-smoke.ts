import { WebSocket as NodeWS } from 'ws';
import { WebsocketProvider } from 'y-websocket';
import * as Y from 'yjs';
import { prisma } from 'db';
import { signSession } from 'protocol';
import { TEXT_KEY } from 'shared/crdt';

/**
 * Faz 3 acceptance smoke. Requires ws-server running on :1234.
 *   pnpm --filter ws-server exec tsx scripts/faz3-smoke.ts
 *
 * Positive: two clients in the same room converge (sync + relay) and see each
 * other's awareness. Negative: connections without a valid token, or for a doc
 * the user doesn't own, are rejected at the handshake (INVARIANT #2).
 */
const WS_URL = 'ws://localhost:1234';
let failures = 0;
function check(name: string, cond: boolean) {
  console.log(`${cond ? 'PASS' : 'FAIL'}  ${name}`);
  if (!cond) failures++;
}
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
async function waitFor(pred: () => boolean, ms = 4000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < ms) {
    if (pred()) return true;
    await sleep(50);
  }
  return pred();
}

/** Raw connect attempt: resolves 'open' | 'rejected'. */
function tryConnect(url: string): Promise<'open' | 'rejected'> {
  return new Promise((resolve) => {
    const ws = new NodeWS(url);
    let settled = false;
    const done = (r: 'open' | 'rejected') => {
      if (settled) return;
      settled = true;
      try {
        ws.close();
      } catch {
        /* ignore */
      }
      resolve(r);
    };
    ws.on('open', () => done('open'));
    ws.on('error', () => done('rejected'));
    ws.on('unexpected-response', () => done('rejected'));
    setTimeout(() => done('rejected'), 3000);
  });
}

async function main() {
  const stamp = Date.now();
  const owner = await prisma.user.create({
    data: { email: `ws_${stamp}@ex.com`, password: 'x' },
    select: { id: true, email: true },
  });
  const stranger = await prisma.user.create({
    data: { email: `wsx_${stamp}@ex.com`, password: 'x' },
    select: { id: true, email: true },
  });
  const doc = await prisma.document.create({
    data: { title: 'WS Room', ownerId: owner.id },
    select: { id: true },
  });

  const ownerToken = await signSession({ id: owner.id, email: owner.email });
  const strangerToken = await signSession({ id: stranger.id, email: stranger.email });

  // ---- Negative: handshake rejections ----
  check('no token → rejected', (await tryConnect(`${WS_URL}/${doc.id}`)) === 'rejected');
  check(
    'valid token but non-owner → rejected',
    (await tryConnect(`${WS_URL}/${doc.id}?token=${strangerToken}`)) === 'rejected',
  );
  check(
    'owner token → accepted',
    (await tryConnect(`${WS_URL}/${doc.id}?token=${ownerToken}`)) === 'open',
  );

  // ---- Positive: two clients converge ----
  const docA = new Y.Doc();
  const docB = new Y.Doc();
  const opts = { params: { token: ownerToken }, WebSocketPolyfill: NodeWS as unknown as typeof WebSocket, connect: true };
  const provA = new WebsocketProvider(WS_URL, doc.id, docA, opts);
  const provB = new WebsocketProvider(WS_URL, doc.id, docB, opts);

  const bothConnected = await waitFor(() => provA.wsconnected && provB.wsconnected, 5000);
  check('both clients connected', bothConnected);

  docA.getText(TEXT_KEY).insert(0, 'Hello CRDT');
  check(
    'edit on A propagates to B',
    await waitFor(() => docB.getText(TEXT_KEY).toString() === 'Hello CRDT'),
  );

  // concurrent edits merge without loss
  docA.getText(TEXT_KEY).insert(0, '[A] ');
  docB.getText(TEXT_KEY).insert(docB.getText(TEXT_KEY).length, ' [B]');
  const converged = await waitFor(
    () => docA.getText(TEXT_KEY).toString() === docB.getText(TEXT_KEY).toString(),
  );
  check('concurrent edits converge on both', converged);
  console.log('   converged text:', JSON.stringify(docA.getText(TEXT_KEY).toString()));

  // ---- Awareness ----
  provA.awareness.setLocalStateField('user', { name: 'Alice', color: '#f00' });
  const bSeesA = await waitFor(() => {
    for (const [, state] of provB.awareness.getStates()) {
      if ((state as { user?: { name?: string } }).user?.name === 'Alice') return true;
    }
    return false;
  });
  check('B sees A awareness (presence)', bSeesA);

  // reload semantics (Faz 3): no persistence — a fresh doc is empty
  provA.destroy();
  provB.destroy();
  await sleep(300);
  const docC = new Y.Doc();
  const provC = new WebsocketProvider(WS_URL, doc.id, docC, opts);
  await waitFor(() => provC.wsconnected, 5000);
  await sleep(500);
  check('after all left, room state is empty (no persistence yet)', docC.getText(TEXT_KEY).toString() === '');
  provC.destroy();

  // cleanup
  await prisma.document.delete({ where: { id: doc.id } });
  await prisma.user.deleteMany({ where: { id: { in: [owner.id, stranger.id] } } });
  await prisma.$disconnect();

  console.log(failures === 0 ? '\nALL PASS' : `\n${failures} FAILED`);
  process.exit(failures === 0 ? 0 : 1);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
