import { NextResponse } from 'next/server';
import { replayTextAt } from '../../../../../lib/audit-service';
import { getAccessibleDocument } from '../../../../../lib/document-service';
import { getSession } from '../../../../../lib/session';

/**
 * Replay endpoint (Faz 6 acceptance): reconstruct the document text as of a given
 * update id — GET .../replay?at=<updateId>. `at=0` yields the base snapshot state.
 * Owner-scoped. 400 on a bad `at`, 404 when the target is outside the surviving
 * (non-pruned) range.
 */
export async function GET(req: Request, { params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const session = await getSession();
  if (!session) return NextResponse.json({ error: 'unauthorized' }, { status: 401 });

  const doc = await getAccessibleDocument(id, session.id);
  if (!doc) return NextResponse.json({ error: 'not found' }, { status: 404 });

  const at = new URL(req.url).searchParams.get('at');
  if (at === null || !/^\d+$/.test(at)) {
    return NextResponse.json({ error: 'invalid `at` (expected a non-negative integer)' }, { status: 400 });
  }

  const result = await replayTextAt(doc.id, BigInt(at));
  if (!result) return NextResponse.json({ error: 'not replayable at that point' }, { status: 404 });

  return NextResponse.json(result);
}
