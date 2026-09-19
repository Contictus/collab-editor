import { renderMarkdown } from './markdown';

/**
 * Document export builders (F12-1). Pure string transforms — Markdown source
 * in, downloadable payloads out. PDF goes through the browser print pipeline
 * (print CSS), so no server round-trip and no new dependency.
 */

export function exportMarkdown(source: string): string {
  return source.endsWith('\n') || source === '' ? source : `${source}\n`;
}

export function exportHtml(title: string, source: string): string {
  const body = renderMarkdown(source);
  const esc = title.replace(/[<>&"]/g, (c) => ({ '<': '&lt;', '>': '&gt;', '&': '&amp;', '"': '&quot;' }[c]!));
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>${esc}</title>
<style>body{font-family:system-ui,sans-serif;max-width:720px;margin:2rem auto;padding:0 1rem;line-height:1.6}table{border-collapse:collapse;width:100%}th,td{border:1px solid #ddd;padding:4px 10px}code{background:#f4f4f4;border-radius:4px;padding:1px 5px}pre{background:#f4f4f4;border-radius:8px;padding:10px 12px;overflow-x:auto}pre code{background:none;padding:0}blockquote{border-left:3px solid #ddd;margin:8px 0;padding:2px 12px;color:#666}@media print{body{margin:0;max-width:none}}</style>
</head>
<body>
${body}
</body>
</html>
`;
}

export function exportFilename(title: string, ext: 'md' | 'html'): string {
  const slug = title
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 60);
  return `${slug || 'document'}.${ext}`;
}
