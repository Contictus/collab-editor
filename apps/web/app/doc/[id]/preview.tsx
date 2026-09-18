'use client';

import { renderMarkdown } from '../../../lib/markdown';

export type PreviewTheme = 'light' | 'dark';

/**
 * Live Markdown preview (F9-2, styled F9-5). Receives the shared Yjs text from
 * the Editor and renders GFM HTML. Read-only — all writes go through
 * CodeMirror + CRDT. GFM table/code/blockquote styling lives here so the
 * editor file stays lean; dark mode is prop-driven (CodeMirror pane itself
 * stays light in v1 — full CM theme is a follow-up).
 */
export function Preview({ text, theme }: { text: string; theme: PreviewTheme }) {
  const dark = theme === 'dark';
  return (
    <>
      <style>{`
        .md-preview table { border-collapse: collapse; width: 100%; margin: 8px 0; font-size: 14px; }
        .md-preview th, .md-preview td { border: 1px solid ${dark ? '#444' : '#ddd'}; padding: 4px 10px; text-align: left; }
        .md-preview thead tr { background: ${dark ? '#2a2a2a' : '#f6f6f6'}; }
        .md-preview code { background: ${dark ? '#2a2a2a' : '#f4f4f4'}; border-radius: 4px; padding: 1px 5px; font-size: 13px; }
        .md-preview pre { background: ${dark ? '#2a2a2a' : '#f4f4f4'}; border-radius: 8px; padding: 10px 12px; overflow-x: auto; }
        .md-preview pre code { background: none; padding: 0; }
        .md-preview blockquote { border-left: 3px solid ${dark ? '#555' : '#ddd'}; margin: 8px 0; padding: 2px 12px; color: ${dark ? '#bbb' : '#666'}; }
        .md-preview ul { padding-left: 22px; }
        .md-preview h1, .md-preview h2, .md-preview h3 { margin: 12px 0 6px; line-height: 1.25; }
      `}</style>
      <div
        data-testid="preview"
        className="md-preview"
        style={{
          border: `1px solid ${dark ? '#444' : '#ddd'}`,
          borderRadius: 8,
          minHeight: 320,
          maxHeight: '60vh',
          padding: '12px 16px',
          background: dark ? '#1e1e1e' : '#fff',
          color: dark ? '#e6e6e6' : '#111',
          overflowY: 'auto',
        }}
        dangerouslySetInnerHTML={{ __html: renderMarkdown(text) }}
      />
    </>
  );
}
