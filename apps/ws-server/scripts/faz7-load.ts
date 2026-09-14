import { WebSocket as NodeWS } from 'ws';
import { WebsocketProvider } from 'y-websocket';
import * as Y from 'yjs';
import { prisma } from 'db';
import { signSession } from 'protocol';
import { TEXT_KEY } from 'shared/crdt';

/**
 * Faz 7 load note (not a graded test — a documented capacity probe). Spins up N
 * Yjs clients in ONE room, each inserts a unique marker roughly simultaneously,
 * and measures how long the single authoritative Y.Doc takes to converge all of
 * them. Confirms the single-node model (INVARIANT #4) holds a small crowd and
 * loses no writes under concurrent load.
 *
 * Run (ws-server must be up):  pnpm --filter ws-server exec tsx scripts/faz7-load.ts [N]
 */
const WS_URL = 'ws://localhost:1234';
const N = Number(process.argv[2] ?? 10);
process.setMaxListeners(N + 10); // each y-websocket provider adds a process exit listener
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

async function waitFor(pred: () => boolean, ms: number): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < ms) {
    if (pred()) return true;
    await sleep(25);
  }
  return pred();
}

async function main() {
  const stamp = Date.now();
  const owner = await prisma.user.create({
    data: { email: `p7load_${stamp}@ex.com`, password: 'x' },
    select: { id: true, email: true },
  });
  const doc = await prisma.document.create({
    data: { title: 'Load Room', ownerId: owner.id },
    select: { id: true },
  });
  const token = await signSession({ id: owner.id, email: owner.email });
  const opts = {
    params: { token },
    WebSocketPolyfill: NodeWS as unknown as typeof WebSocket,
    connect: true,
  };

  const markers = Array.from({ length: N }, (_, i) => `<c${i}>`);
  const clients = markers.map(() => {
    const ydoc = new Y.Doc();
    const provider = new WebsocketProvider(WS_URL, doc.id, ydoc, opts);
    return { ydoc, provider };
  });

  // Wait for all sockets connected.
  const connected = await waitFor(() => clients.every((c) => c.provider.wsconnected), 15_000);
  console.log(`${N} clients connected: ${connected}`);

  // Fire all inserts as close to simultaneously as possible.
  const t0 = Date.now();
  clients.forEach((c, i) => c.ydoc.getText(TEXT_KEY).insert(0, markers[i]!));

  // Converged = every client's text contains every marker (order is CRDT-decided).
  const allSeeAll = () =>
    clients.every((c) => {
      const text = c.ydoc.getText(TEXT_KEY).toString();
      return markers.every((m) => text.includes(m));
    });
  const converged = await waitFor(allSeeAll, 20_000);
  const elapsed = Date.now() - t0;

  const final = clients[0]!.ydoc.getText(TEXT_KEY).toString();
  const identical = clients.every((c) => c.ydoc.getText(TEXT_KEY).toString() === final);
  const totalChars = markers.join('').length;

  console.log('— load note —');
  console.log(`clients:            ${N}`);
  console.log(`converged:          ${converged} in ${elapsed}ms`);
  console.log(`all identical:      ${identical}`);
  console.log(`no lost writes:     ${final.length === totalChars} (chars ${final.length}/${totalChars})`);
  console.log(`final length:       ${final.length}`);

  clients.forEach((c) => c.provider.destroy());
  await sleep(1000); // let the server take a final snapshot before we delete
  await prisma.documentUpdate.deleteMany({ where: { documentId: doc.id } });
  await prisma.documentSnapshot.deleteMany({ where: { documentId: doc.id } });
  await prisma.document.delete({ where: { id: doc.id } });
  await prisma.user.delete({ where: { id: owner.id } });
  await prisma.$disconnect();

  const ok = connected && converged && identical && final.length === totalChars;
  console.log(ok ? '\nLOAD OK' : '\nLOAD DEGRADED');
  process.exit(ok ? 0 : 1);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
