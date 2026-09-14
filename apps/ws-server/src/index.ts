import { createServer, type IncomingMessage } from 'node:http';
import type { Duplex } from 'node:stream';
import { WebSocketServer } from 'ws';
import { authenticate, authorizeDocument, parseUpgrade } from './auth';
import { getRoom, roomStats, setupConnection } from './sync';

/**
 * Standalone WebSocket server — separate process from Next (ADR 0002).
 *
 * Faz 3: every connection passes JWT auth + document authorization during the
 * HTTP `upgrade`, BEFORE joining a room (INVARIANT #2). On success it joins the
 * in-memory Y.Doc room and speaks y-protocols sync + awareness (sync.ts).
 * Persistence (load-on-open, op-log append, snapshots) is Faz 4.
 */

const PORT = Number(process.env.WS_PORT ?? 1234);

/**
 * Hardening: cap a single WS frame. Yjs messages are incremental sync/awareness
 * updates plus one full-state sync on join — all well under this for a Markdown
 * doc. `ws` enforces it and closes the socket with 1009 (Message Too Big) if a
 * client sends a larger frame, bounding memory per connection. Env-overridable.
 */
const MAX_PAYLOAD = Number(process.env.WS_MAX_PAYLOAD ?? 1_000_000); // 1 MB

const server = createServer((req, res) => {
  if (req.url === '/health') {
    res.writeHead(200, { 'content-type': 'application/json' });
    res.end(JSON.stringify({ status: 'ok', service: 'ws-server', ...roomStats() }));
    return;
  }
  res.writeHead(404);
  res.end();
});

const wss = new WebSocketServer({ noServer: true, maxPayload: MAX_PAYLOAD });

function reject(socket: Duplex, code: number, reason: string): void {
  socket.write(`HTTP/1.1 ${code} ${reason}\r\nConnection: close\r\n\r\n`);
  socket.destroy();
}

async function handleUpgrade(req: IncomingMessage, socket: Duplex, head: Buffer): Promise<void> {
  const { docId, token } = parseUpgrade(req);
  if (!docId || !token) return reject(socket, 401, 'Unauthorized');

  const user = await authenticate(token);
  if (!user) return reject(socket, 401, 'Unauthorized');

  if (!(await authorizeDocument(docId, user.id))) return reject(socket, 403, 'Forbidden');

  // Load (or reuse) the authoritative room before accepting the socket, so its
  // message listeners are attached synchronously on open — no lost first message.
  const room = await getRoom(docId);
  wss.handleUpgrade(req, socket, head, (ws) => setupConnection(ws, room, user));
}

server.on('upgrade', (req, socket, head) => {
  handleUpgrade(req, socket, head).catch((err) => {
    console.error('[ws] upgrade error', err);
    reject(socket, 500, 'Internal Server Error');
  });
});

server.listen(PORT, () => {
  console.log(`[ws-server] listening on :${PORT} (http /health, ws y-protocols sync+awareness)`);
});
