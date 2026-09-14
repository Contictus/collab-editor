import * as decoding from 'lib0/decoding';
import * as encoding from 'lib0/encoding';
import type { WebSocket } from 'ws';
import * as awarenessProtocol from 'y-protocols/awareness';
import * as syncProtocol from 'y-protocols/sync';
import * as Y from 'yjs';
import { MessageType } from 'protocol';
import { SNAPSHOT_THRESHOLD, type SessionUser } from 'shared';
import { appendUpdate, compact, loadInto } from './persistence';

/**
 * y-protocols sync + awareness over ws (Faz 3) + persistence (Faz 4).
 *
 * Each docId has ONE authoritative in-memory Y.Doc (INVARIANT #4). The doc is
 * loaded from the op log on first open, every applied update is appended to the
 * append-only log, and the log is compacted into a snapshot every N updates.
 * A final snapshot is taken when the last client leaves before the doc is dropped.
 */

/** Compaction threshold — env-overridable for testing; defaults to data-model N=100. */
const THRESHOLD = Number(process.env.WS_SNAPSHOT_THRESHOLD ?? SNAPSHOT_THRESHOLD);

interface Room {
  docId: string;
  doc: Y.Doc;
  awareness: awarenessProtocol.Awareness;
  conns: Map<WebSocket, Set<number>>;
  clock: number;
  sinceSnapshot: number;
  /** Serializes DB writes so clock stays monotonic and compaction is consistent. */
  writeChain: Promise<void>;
}

const rooms = new Map<string, Room>();
const loading = new Map<string, Promise<Room>>();

function send(conn: WebSocket, message: Uint8Array): void {
  if (conn.readyState !== 1) {
    conn.close();
    return;
  }
  try {
    conn.send(message);
  } catch {
    conn.close();
  }
}

function broadcast(room: Room, message: Uint8Array, except?: WebSocket): void {
  room.conns.forEach((_ids, conn) => {
    if (conn !== except) send(conn, message);
  });
}

function persistUpdate(room: Room, update: Uint8Array): void {
  room.writeChain = room.writeChain
    .then(async () => {
      await appendUpdate(room.docId, update, room.clock++);
      room.sinceSnapshot++;
      if (room.sinceSnapshot >= THRESHOLD) {
        await compact(room.docId, room.doc);
        room.sinceSnapshot = 0;
      }
    })
    .catch((err) => console.error(`[persist] error (doc ${room.docId})`, err));
}

function onDocUpdate(room: Room, update: Uint8Array, origin: unknown): void {
  const encoder = encoding.createEncoder();
  encoding.writeVarUint(encoder, MessageType.Sync);
  syncProtocol.writeUpdate(encoder, update);
  const message = encoding.toUint8Array(encoder);
  broadcast(room, message, origin instanceof Object ? (origin as WebSocket) : undefined);
  // Persist every state-mutating update (append-only op log).
  persistUpdate(room, update);
}

function onAwarenessUpdate(
  room: Room,
  changes: { added: number[]; updated: number[]; removed: number[] },
  origin: unknown,
): void {
  const changed = [...changes.added, ...changes.updated, ...changes.removed];
  if (origin && room.conns.has(origin as WebSocket)) {
    const ids = room.conns.get(origin as WebSocket)!;
    changes.added.forEach((id) => ids.add(id));
    changes.removed.forEach((id) => ids.delete(id));
  }
  const encoder = encoding.createEncoder();
  encoding.writeVarUint(encoder, MessageType.Awareness);
  encoding.writeVarUint8Array(
    encoder,
    awarenessProtocol.encodeAwarenessUpdate(room.awareness, changed),
  );
  // Awareness is ephemeral — broadcast only, never persisted.
  broadcast(room, encoding.toUint8Array(encoder));
}

/** Get or create+load the authoritative room for a document (deduped concurrently). */
export async function getRoom(docId: string): Promise<Room> {
  const existing = rooms.get(docId);
  if (existing) return existing;
  const pending = loading.get(docId);
  if (pending) return pending;

  const build = (async (): Promise<Room> => {
    const doc = new Y.Doc();
    const awareness = new awarenessProtocol.Awareness(doc);
    awareness.setLocalState(null);

    // Load BEFORE attaching the update listener so replay doesn't re-append.
    const { clock, sinceSnapshot } = await loadInto(doc, docId);

    const room: Room = {
      docId,
      doc,
      awareness,
      conns: new Map(),
      clock,
      sinceSnapshot,
      writeChain: Promise.resolve(),
    };

    doc.on('update', (update: Uint8Array, origin: unknown) => onDocUpdate(room, update, origin));
    awareness.on(
      'update',
      (changes: { added: number[]; updated: number[]; removed: number[] }, origin: unknown) =>
        onAwarenessUpdate(room, changes, origin),
    );

    rooms.set(docId, room);
    loading.delete(docId);
    return room;
  })();

  loading.set(docId, build);
  return build;
}

function onMessage(conn: WebSocket, room: Room, data: Uint8Array): void {
  const decoder = decoding.createDecoder(data);
  const type = decoding.readVarUint(decoder);

  switch (type) {
    case MessageType.Sync: {
      const encoder = encoding.createEncoder();
      encoding.writeVarUint(encoder, MessageType.Sync);
      syncProtocol.readSyncMessage(decoder, encoder, room.doc, conn);
      if (encoding.length(encoder) > 1) send(conn, encoding.toUint8Array(encoder));
      break;
    }
    case MessageType.Awareness: {
      awarenessProtocol.applyAwarenessUpdate(
        room.awareness,
        decoding.readVarUint8Array(decoder),
        conn,
      );
      break;
    }
    default:
      break;
  }
}

async function finalizeRoom(room: Room): Promise<void> {
  await room.writeChain; // flush pending appends/compactions
  if (room.conns.size > 0) return; // someone rejoined during the flush
  try {
    // Final checkpoint so the next open loads fast (data-model: snapshot on last leave).
    await compact(room.docId, room.doc);
  } catch (err) {
    console.error(`[persist] final snapshot failed (doc ${room.docId})`, err);
  }
  if (room.conns.size > 0) return;
  room.awareness.destroy();
  room.doc.destroy();
  rooms.delete(room.docId);
}

function closeConnection(conn: WebSocket, room: Room): void {
  const controlled = room.conns.get(conn);
  room.conns.delete(conn);
  if (controlled) {
    awarenessProtocol.removeAwarenessStates(room.awareness, [...controlled], null);
  }
  if (room.conns.size === 0) void finalizeRoom(room);
}

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export function setupConnection(conn: WebSocket, room: Room, _user: SessionUser): void {
  room.conns.set(conn, new Set());

  conn.on('message', (data: Buffer) =>
    onMessage(conn, room, new Uint8Array(data.buffer, data.byteOffset, data.byteLength)),
  );
  conn.on('close', () => closeConnection(conn, room));
  conn.on('error', (err) => console.error(`[ws] conn error (doc ${room.docId})`, err));

  const syncEncoder = encoding.createEncoder();
  encoding.writeVarUint(syncEncoder, MessageType.Sync);
  syncProtocol.writeSyncStep1(syncEncoder, room.doc);
  send(conn, encoding.toUint8Array(syncEncoder));

  const states = room.awareness.getStates();
  if (states.size > 0) {
    const awarenessEncoder = encoding.createEncoder();
    encoding.writeVarUint(awarenessEncoder, MessageType.Awareness);
    encoding.writeVarUint8Array(
      awarenessEncoder,
      awarenessProtocol.encodeAwarenessUpdate(room.awareness, [...states.keys()]),
    );
    send(conn, encoding.toUint8Array(awarenessEncoder));
  }
}

export function roomStats(): { rooms: number; connections: number } {
  let connections = 0;
  rooms.forEach((r) => (connections += r.conns.size));
  return { rooms: rooms.size, connections };
}
