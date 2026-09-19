import { describe, expect, it } from 'vitest';
import { exportFilename, exportHtml, exportMarkdown } from './export';

describe('export builders (F12-1)', () => {
  it('keeps markdown source with trailing newline', () => {
    expect(exportMarkdown('# hi')).toBe('# hi\n');
    expect(exportMarkdown('')).toBe('');
  });

  it('wraps rendered html in a printable shell', () => {
    const html = exportHtml('My Doc', '# hi');
    expect(html).toContain('<h1>');
    expect(html).toContain('<title>My Doc</title>');
    expect(html).toContain('@media print');
  });

  it('escapes titles and slugifies filenames', () => {
    expect(exportHtml('<x>', 'a')).toContain('<title>&lt;x&gt;</title>');
    expect(exportFilename('Hello World!', 'md')).toBe('hello-world.md');
    expect(exportFilename('', 'html')).toBe('document.html');
  });
});
