import type { ReactNode } from 'react';

export const metadata = {
  title: 'Collab Editor',
  description: 'Real-time collaborative Markdown editor (CRDT / Yjs)',
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body
        style={{
          fontFamily: 'system-ui, -apple-system, Segoe UI, Roboto, sans-serif',
          margin: 0,
          background: '#f8f7f5',
          color: '#111',
        }}
      >
        <header
          style={{
            position: 'sticky',
            top: 0,
            zIndex: 10,
            backdropFilter: 'blur(8px)',
            background: '#ffffffcc',
            borderBottom: '1px solid #e8e8e8',
          }}
        >
          <div
            style={{
              maxWidth: 960,
              margin: '0 auto',
              padding: '12px 24px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 12,
            }}
          >
            <a href="/" style={{ fontWeight: 800, letterSpacing: -0.5, textDecoration: 'none', color: '#111' }}>
              collab-editor
            </a>
            <span style={{ fontSize: 12, color: '#888', letterSpacing: 0.3, textTransform: 'uppercase' }}>
              Yjs · CRDT · live Markdown
            </span>
          </div>
        </header>
        <div style={{ maxWidth: 960, margin: '0 auto', padding: 24 }}>{children}</div>
      </body>
    </html>
  );
}
