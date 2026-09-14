import Link from 'next/link';
import { notFound } from 'next/navigation';
import { getAccessibleDocument, listCollaborators } from '../../../lib/document-service';
import { requireSession } from '../../../lib/session';
import { displayName, userColor } from '../../../lib/user-color';
import { CopyLink } from './copy-link';
import { DocControls } from './doc-controls';
import { Editor } from './editor';
import { ShareControls } from './share-controls';

/**
 * Document view (Faz 5) — Server Component gate + live editor.
 * Owner-scoped: unknown/non-owned id → 404. The client Editor takes over: it
 * connects to the ws-server, loads state via sync, and renders CodeMirror with
 * remote cursors. SSR just renders the shell (auth + title).
 */
export default async function DocumentPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const user = await requireSession();

  const doc = await getAccessibleDocument(id, user.id);
  if (!doc) notFound();

  const collaborators = doc.isOwner ? await listCollaborators(doc.id) : [];

  return (
    <main style={{ display: 'grid', gap: 14 }}>
      <p style={{ display: 'flex', gap: 12, margin: 0, fontSize: 13 }}>
        <Link href="/documents" style={{ textDecoration: 'none', color: '#555' }}>
          ← Documents
        </Link>
        <Link href={`/doc/${doc.id}/history`} style={{ textDecoration: 'none', color: '#111', fontWeight: 600 }}>
          History / replay →
        </Link>
        <span style={{ marginLeft: 'auto' }}>
          <CopyLink />
        </span>
      </p>
      <h1 style={{ margin: 0, fontSize: 22, letterSpacing: -0.5 }}>{doc.title}</h1>
      {doc.isOwner ? (
        <>
          <DocControls docId={doc.id} title={doc.title} />
          <ShareControls docId={doc.id} collaborators={collaborators} />
        </>
      ) : (
        <p style={{ color: '#777', margin: 0, fontSize: 13, padding: '8px 10px', border: '1px solid #e8e8e8', borderRadius: 8, background: '#fafaf7' }}>
          Shared with you · collaborator · live sync active
        </p>
      )}
      <Editor docId={doc.id} userName={displayName(user.email)} userColor={userColor(user.email)} />
      <p style={{ margin: 0, color: '#999', fontSize: 11 }}>
        Invite via Share, open in second window to see cursors. History keeps op-log snapshots for replay.
      </p>
    </main>
  );
}
