import { describe, expect, it } from 'vitest';
import { renderMarkdown } from './markdown';

describe('renderMarkdown (F9-1)', () => {
  it('renders headings and bold', () => {
    const html = renderMarkdown('# Title\n\nHello **world**');
    expect(html).toContain('<h1>');
    expect(html).toContain('<strong>world</strong>');
  });

  it('renders GFM tables and task lists', () => {
    const html = renderMarkdown('| a | b |\n|---|---|\n| 1 | 2 |\n\n- [ ] todo\n- [x] done');
    expect(html).toContain('<table>');
    expect(html).toContain('type="checkbox"');
  });

  it('returns empty string for empty input', () => {
    expect(renderMarkdown('')).toBe('');
  });
});
