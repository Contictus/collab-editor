import type { IncomingMessage } from 'node:http';
import { prisma } from 'db';
import { SESSION_COOKIE, verifySession } from 'protocol';
import type { SessionUser } from 'shared';

/**
 * Handshake auth (INVARIANT #2). Runs during the HTTP `upgrade`, before the
 * socket joins any room or a single sync/awareness byte is processed.
 *
 * docId source: `?doc=<id>` (our documented form) or the URL path segment
 * (y-websocket client puts the room name in the path).
 * token source: `?token=<jwt>` or the session cookie (same-domain browser).
 */
export function parseUpgrade(req: IncomingMessage): { docId: string | null; token: string | null } {
  const url = new URL(req.url ?? '', 'http://localhost');
  const docId = url.searchParams.get('doc') ?? decodeURIComponent(url.pathname.replace(/^\/+/, '')) ?? null;
  const token = url.searchParams.get('token') ?? cookieToken(req);
  return { docId: docId || null, token };
}

function cookieToken(req: IncomingMessage): string | null {
  const raw = req.headers.cookie;
  if (!raw) return null;
  for (const part of raw.split(';')) {
    const [name, ...rest] = part.trim().split('=');
    if (name === SESSION_COOKIE) return decodeURIComponent(rest.join('='));
  }
  return null;
}

export async function authenticate(token: string): Promise<SessionUser | null> {
  return verifySession(token);
}

/**
 * Handshake authorization: owner OR collaborator (Faz 8). Same predicate the web
 * app enforces on REST reads (getAccessibleDocument) — REST and live sync share
 * one access model. An unshared, non-owned document → 403 before any room join.
 */
export async function authorizeDocument(docId: string, userId: string): Promise<boolean> {
  const doc = await prisma.document.findFirst({
    where: { id: docId, OR: [{ ownerId: userId }, { collaborators: { some: { userId } } }] },
    select: { id: true },
  });
  return doc !== null;
}
