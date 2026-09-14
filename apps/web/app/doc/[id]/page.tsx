import Link from 'next/link';
import { notFound } from 'next/navigation';
import { getAccessibleDocument, listCollaborators } from '../../../lib/document-service';
import { requireSession } from '../../../lib/session';
import { displayName, userColor } from '../../../lib/user-color';
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
    <main>
      <p style={{ display: 'flex', gap: 16 }}>
        <Link href="/documents">← Documents</Link>
        <Link href={`/doc/${doc.id}/history`}>History / replay →</Link>
      </p>
      <h1>{doc.title}</h1>
      {doc.isOwner ? (
        <>
          <DocControls docId={doc.id} title={doc.title} />
          <ShareControls docId={doc.id} collaborators={collaborators} />
        </>
      ) : (
        <p style={{ color: '#777', margin: '4px 0' }}>Shared with you · collaborator</p>
      )}
      <Editor docId={doc.id} userName={displayName(user.email)} userColor={userColor(user.email)} />
    </main>
  );
}
