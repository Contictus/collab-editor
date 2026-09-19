'use client';

import { useState } from 'react';
import { exportFilename, exportHtml, exportMarkdown } from '../../../lib/export';

/**
 * Document export (F12-3). Markdown/HTML download as blobs; PDF through the
 * browser print pipeline on a standalone printable window. Client-only, works
 * off the live Yjs text — no server round-trip.
 */
export function ExportButtons({ title, text }: { title: string; text: string }) {
  const [error, setError] = useState<string | null>(null);

  function download(payload: string, mime: string, filename: string) {
    const blob = new Blob([payload], { type: `${mime};charset=utf-8` });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }

  function printPdf() {
    const win = window.open('', '_blank', 'width=800,height=600');
    if (!win) {
      setError('Popup blocked — allow popups to print.');
      return;
    }
    setError(null);
    win.document.write(exportHtml(title, text));
    win.document.close();
    win.focus();
    win.print();
  }

  const btn = {
    border: '1px solid #ddd',
    borderRadius: 6,
    background: '#fff',
    padding: '4px 10px',
    fontSize: 13,
    cursor: 'pointer',
  } as const;

  return (
    <span style={{ display: 'inline-flex', gap: 6, alignItems: 'center' }}>
      <button
        type="button"
        data-testid="export-md"
        title="Download Markdown"
        onClick={() => download(exportMarkdown(text), 'text/markdown', exportFilename(title, 'md'))}
        style={btn}
      >
        .md
      </button>
      <button
        type="button"
        data-testid="export-html"
        title="Download HTML"
        onClick={() => download(exportHtml(title, text), 'text/html', exportFilename(title, 'html'))}
        style={btn}
      >
        .html
      </button>
      <button type="button" data-testid="export-pdf" title="Print / save as PDF" onClick={printPdf} style={btn}>
        PDF
      </button>
      {error && <span style={{ color: 'crimson', fontSize: 12 }}>{error}</span>}
    </span>
  );
}
