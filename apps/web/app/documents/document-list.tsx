'use client';

import Link from 'next/link';
import { useState } from 'react';
import type { DocumentSummary } from '../../lib/document-service';

/**
 * Filterable document grid (F12-24). Client-side title filter over the
 * server-loaded list — no query round-trip, instant feedback.
 */
export function DocumentList({ docs }: { docs: DocumentSummary[] }) {
  const [q, setQ] = useState('');
  const filtered = docs.filter((d) => d.title.toLowerCase().includes(q.trim().toLowerCase()));

  return (
    <div style={{ display: 'grid', gap: 12 }}>
      {docs.length > 3 && (
        <input
          data-testid="doc-search"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search documents…"
          style={{ padding: '8px 12px', border: '1px solid #ddd', borderRadius: 8, fontSize: 14 }}
        />
      )}
      {filtered.length === 0 ? (
        <p data-testid="doc-empty" style={{ color: '#666', border: '1px dashed #ddd', borderRadius: 12, padding: 16, background: '#fff', margin: 0 }}>
          {docs.length === 0
            ? 'No documents yet. Create one above — it becomes a Yjs room on first open.'
            : `No documents match "${q}".`}
        </p>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))', gap: 12 }}>
          {filtered.map((d) => (
            <Link
              key={d.id}
              href={`/doc/${d.id}`}
              style={{
                display: 'block',
                textDecoration: 'none',
                color: 'inherit',
                border: '1px solid #e8e8e8',
                borderRadius: 12,
                padding: 14,
                background: '#fff',
                boxShadow: '0 1px 4px #00000006',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                <div style={{ fontWeight: 600, flex: 1, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                  {d.title}
                </div>
                {!d.isOwner && (
                  <span
                    style={{
                      fontSize: 10,
                      letterSpacing: 0.3,
                      textTransform: 'uppercase',
                      padding: '2px 6px',
                      borderRadius: 999,
                      background: '#eaf0ff',
                      color: '#3b5bdb',
                      border: '1px solid #dbe4ff',
                    }}
                  >
                    shared
                  </span>
                )}
              </div>
              <div style={{ fontSize: 12, color: '#999' }}>{d.updatedAt.toLocaleString()}</div>
              <div style={{ marginTop: 8, fontSize: 12, color: '#111', fontWeight: 600 }}>Open →</div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
