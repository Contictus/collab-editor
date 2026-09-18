'use client';

import { renderMarkdown } from '../../../lib/markdown';

/**
 * Live Markdown preview (F9-2). Receives the shared Yjs text from the Editor
 * and renders GFM HTML. Read-only — all writes go through CodeMirror + CRDT.
 */
export function Preview({ text }: { text: string }) {
  return (
    <div
      data-testid="preview"
      className="md-preview"
      style={{
        border: '1px solid #ddd',
        borderRadius: 8,
        minHeight: 320,
        padding: '12px 16px',
        background: '#fff',
        overflowY: 'auto',
      }}
      dangerouslySetInnerHTML={{ __html: renderMarkdown(text) }}
    />
  );
}
