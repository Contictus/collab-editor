import Link from 'next/link';
import { getSession } from '../lib/session';

export default async function HomePage() {
  const session = await getSession();
  return (
    <main style={{ display: 'grid', gap: 24 }}>
      <section
        style={{
          padding: 28,
          border: '1px solid #e8e8e8',
          borderRadius: 16,
          background: 'linear-gradient(180deg, #fff, #fafaf7)',
          boxShadow: '0 6px 24px #00000008',
        }}
      >
        <div style={{ fontSize: 12, letterSpacing: 0.4, textTransform: 'uppercase', color: '#888' }}>
          Real-time · CRDT · no lost writes
        </div>
        <h1 style={{ margin: '8px 0 8px', fontSize: 36, lineHeight: 1.1, letterSpacing: -1 }}>Collab Editor</h1>
        <p style={{ margin: 0, color: '#555', maxWidth: 620, lineHeight: 1.6 }}>
          Markdown where many cursors type at once. Yjs <code>Y.Text</code> merges edits deterministically, a
          standalone ws-server handles sync + awareness, Postgres keeps an append-only op log with snapshots.
        </p>
        <div style={{ display: 'flex', gap: 10, marginTop: 16, flexWrap: 'wrap' }}>
          {session ? (
            <>
              <span style={{ padding: '6px 10px', border: '1px solid #e8e8e8', borderRadius: 999, background: '#fff', fontSize: 13 }}>
                {session.email}
              </span>
              <Link
                href="/documents"
                style={{
                  padding: '8px 14px',
                  borderRadius: 999,
                  background: '#111',
                  color: '#fff',
                  textDecoration: 'none',
                  fontSize: 13,
                  fontWeight: 600,
                }}
              >
                Open documents →
              </Link>
            </>
          ) : (
            <>
              <Link
                href="/login"
                style={{
                  padding: '8px 14px',
                  borderRadius: 999,
                  background: '#111',
                  color: '#fff',
                  textDecoration: 'none',
                  fontSize: 13,
                  fontWeight: 600,
                }}
              >
                Sign in
              </Link>
              <Link
                href="/register"
                style={{
                  padding: '8px 14px',
                  borderRadius: 999,
                  border: '1px solid #ddd',
                  background: '#fff',
                  textDecoration: 'none',
                  fontSize: 13,
                  color: '#111',
                }}
              >
                Create account
              </Link>
            </>
          )}
        </div>
      </section>

      <section style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 12 }}>
        {[
          { k: 'CRDT', v: 'Yjs Y.Text — ops commute, offline edits merge on reconnect.' },
          { k: 'Sync', v: 'y-protocols over ws — auth before first message (INVARIANT #2).' },
          { k: 'Durability', v: 'Op-log + snapshot — replay at any point, audit trail.' },
        ].map((c) => (
          <div key={c.k} style={{ border: '1px solid #e8e8e8', borderRadius: 12, padding: 14, background: '#fff' }}>
            <div style={{ fontSize: 11, letterSpacing: 0.4, textTransform: 'uppercase', color: '#888' }}>{c.k}</div>
            <div style={{ marginTop: 6, fontSize: 13, color: '#333', lineHeight: 1.5 }}>{c.v}</div>
          </div>
        ))}
      </section>

      <p style={{ color: '#999', fontSize: 12, margin: 0 }}>
        Tip: open a document in two windows — edits and cursors sync live. Try <code>History / replay</code> after a few edits.
      </p>
    </main>
  );
}
