import { listDocuments } from '../../lib/document-service';
import { requireSession } from '../../lib/session';
import { logoutAction } from '../actions/auth';
import { CreateDocumentForm } from './create-form';
import { DocumentList } from './document-list';

/**
 * Protected documents list (Faz 2). Owner-scoped: requireSession then list only
 * this user's documents.
 */
export default async function DocumentsPage() {
  const user = await requireSession();
  const docs = await listDocuments(user.id);

  return (
    <main style={{ display: 'grid', gap: 16 }}>
      <header
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          gap: 12,
          padding: '12px 14px',
          border: '1px solid #e8e8e8',
          borderRadius: 12,
          background: '#fff',
        }}
      >
        <div>
          <h1 style={{ margin: 0, fontSize: 20, letterSpacing: -0.5 }}>Documents</h1>
          <div style={{ fontSize: 12, color: '#888' }}>
            {docs.length} {docs.length === 1 ? 'document' : 'documents'} · {user.email}
          </div>
        </div>
        <form action={logoutAction}>
          <button
            type="submit"
            style={{ padding: '6px 10px', borderRadius: 999, border: '1px solid #ddd', background: '#fff', fontSize: 12 }}
          >
            Sign out
          </button>
        </form>
      </header>

      <div style={{ border: '1px solid #e8e8e8', borderRadius: 12, padding: 14, background: '#fff' }}>
        <div style={{ fontSize: 11, letterSpacing: 0.4, textTransform: 'uppercase', color: '#888', marginBottom: 8 }}>
          New document
        </div>
        <CreateDocumentForm />
      </div>

      {docs.length === 0 ? (
        <p style={{ color: '#666', border: '1px dashed #ddd', borderRadius: 12, padding: 16, background: '#fff', margin: 0 }}>
          No documents yet. Create one above — it becomes a Yjs room on first open.
        </p>
      ) : (
        <DocumentList docs={docs} />
      )}
    </main>
  );
}
