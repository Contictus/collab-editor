/**
 * Pure Markdown text transforms for the editor toolbar (F9-3). Pure string
 * functions so they stay unit-testable; toolbar.tsx maps them onto the
 * CodeMirror transaction.
 */

export interface WrapResult {
  text: string;
  start: number;
  end: number;
}

/** Wrap [start,end) with before/after. Empty selection → placeholder selected. */
export function wrapSelection(
  text: string,
  start: number,
  end: number,
  before: string,
  after: string,
  placeholder = 'text',
): WrapResult {
  if (start === end) {
    const insert = `${before}${placeholder}${after}`;
    const next = text.slice(0, start) + insert + text.slice(end);
    return { text: next, start: start + before.length, end: start + before.length + placeholder.length };
  }
  const selected = text.slice(start, end);
  const next = text.slice(0, start) + before + selected + after + text.slice(end);
  return { text: next, start: start + before.length, end: end + before.length };
}

/** Prefix every line intersecting [start,end) with prefix (headings, lists, tasks). */
export function prefixLines(text: string, start: number, end: number, prefix: string): WrapResult {
  const lineStart = text.lastIndexOf('\n', start - 1) + 1;
  let lineEnd = text.indexOf('\n', end);
  if (lineEnd === -1) lineEnd = text.length;
  const block = text.slice(lineStart, lineEnd);
  const lines = block.split('\n');
  const prefixed = lines.map((l) => (l.startsWith(prefix) ? l : prefix + l)).join('\n');
  const next = text.slice(0, lineStart) + prefixed + text.slice(lineEnd);
  const added = prefixed.length - block.length;
  return { text: next, start, end: end + added };
}

export const TABLE_SNIPPET = '| a | b |\n|---|---|\n| 1 | 2 |\n';
export const TASK_SNIPPET = '- [ ] ';
export const CODE_SNIPPET = { before: '```\n', after: '\n```' };
