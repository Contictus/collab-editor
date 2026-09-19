import { notFound } from 'next/navigation';
import { getPublicDocument, loadInitialText } from '../../../lib/document-service';
import { renderMarkdown } from '../../../lib/markdown';

/**
 * Public read-only document (F12-23). No session — the token in the URL is the
 * capability. Renders the latest persisted text; no live sync, no editor.
 */
export default async function PublicPage({ params }: { params: Promise<{ token: string }> }) {
  const { token } = await params;
  const doc = await getPublicDocument(token);
  if (!doc) notFound();

  const text = await loadInitialText(doc.id);

  return (
    <main style={{ display: 'grid', gap: 14, maxWidth: 720, margin: '0 auto' }}>
      <h1 style={{ margin: 0, fontSize: 24, letterSpacing: -0.5 }}>{doc.title}</h1>
      <p data-testid="public-badge" style={{ margin: 0, fontSize: 12, color: '#999' }}>
        Public read-only copy
      </p>
      <article
        data-testid="public-text"
        style={{ border: '1px solid #e8e8e8', borderRadius: 8, padding: '12px 16px', background: '#fff', lineHeight: 1.6 }}
        dangerouslySetInnerHTML={{ __html: renderMarkdown(text) }}
      />
    </main>
  );
}
