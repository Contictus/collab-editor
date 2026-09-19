import 'server-only';
import { cookies } from 'next/headers';
import { SESSION_COOKIE } from 'protocol';
import { getAccessibleDocument } from './document-service';
import { requireSession } from './session';

/**
 * History restore via the Go sync server (F11-11). Restore must apply through
 * the live room (broadcast + op-log append flow untouched), so the web layer
 * forwards the session cookie to the Go restore endpoint — same identity,
 * same access gate (owner OR collaborator).
 */

export class RestoreError extends Error {}

const GO_API_URL = process.env.GO_API_URL ?? 'http://localhost:8080';

export async function restoreAt(documentId: string, at: string): Promise<string> {
  const user = await requireSession();
  const doc = await getAccessibleDocument(documentId, user.id);
  if (!doc) throw new RestoreError('Not found.');
  if (!/^\d+$/.test(at)) throw new RestoreError('Invalid restore point.');

  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) throw new RestoreError('Not signed in.');

  const res = await fetch(`${GO_API_URL}/api/documents/${documentId}/restore`, {
    method: 'POST',
    headers: { 'content-type': 'application/json', cookie: `${SESSION_COOKIE}=${token}` },
    body: JSON.stringify({ at }),
  });
  if (res.status === 404) throw new RestoreError('Not replayable at that point (pruned).');
  if (!res.ok) throw new RestoreError(`Restore failed (${res.status}).`);
  const data = (await res.json()) as { text: string };
  return data.text;
}
