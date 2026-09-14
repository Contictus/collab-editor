import Link from 'next/link';
import { listDocuments } from '../../lib/document-service';
import { requireSession } from '../../lib/session';
import { logoutAction } from '../actions/auth';
import { CreateDocumentForm } from './create-form';

/**
 * Protected documents list (Faz 2). Owner-scoped: requireSession then list only
 * this user's documents.
 */
export default async function DocumentsPage() {
  const user = await requireSession();
  const docs = await listDocuments(user.id);

  return (
    <main>
      <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}>
        <h1>Documents</h1>
        <form action={logoutAction}>
          <span style={{ marginRight: 8, color: '#666' }}>{user.email}</span>
          <button type="submit">Sign out</button>
        </form>
      </header>

      <CreateDocumentForm />

      {docs.length === 0 ? (
        <p style={{ color: '#666' }}>No documents yet. Create one above.</p>
      ) : (
        <ul>
          {docs.map((d) => (
            <li key={d.id}>
              <Link href={`/doc/${d.id}`}>{d.title}</Link>{' '}
              {!d.isOwner && (
                <small style={{ color: '#3b7dd8', marginRight: 6 }}>shared</small>
              )}
              <small style={{ color: '#999' }}>{d.updatedAt.toLocaleString()}</small>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
