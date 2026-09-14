import { NextResponse } from 'next/server';
import { getAuditLog } from '../../../../../lib/audit-service';
import { getAccessibleDocument } from '../../../../../lib/document-service';
import { getSession } from '../../../../../lib/session';

/**
 * Audit endpoint (Faz 6): the persisted history of a document — update-log counts,
 * snapshot checkpoints, and the surviving replayable window. Owner-scoped:
 * unauthenticated → 401, non-owned/unknown id → 404.
 */
export async function GET(_req: Request, { params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const session = await getSession();
  if (!session) return NextResponse.json({ error: 'unauthorized' }, { status: 401 });

  const doc = await getAccessibleDocument(id, session.id);
  if (!doc) return NextResponse.json({ error: 'not found' }, { status: 404 });

  return NextResponse.json(await getAuditLog(doc.id, doc.title));
}
