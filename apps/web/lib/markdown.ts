import { marked } from 'marked';

/**
 * Markdown → HTML for the live preview pane (F9-1). Pure string transform so it
 * stays unit-testable in node (no DOM). GFM tables / task lists are on; the
 * source is the shared Yjs text, already trusted (authenticated collaborators).
 */
marked.setOptions({ gfm: true, breaks: true });

export function renderMarkdown(src: string): string {
  const html = marked.parse(src);
  return typeof html === 'string' ? html : '';
}
