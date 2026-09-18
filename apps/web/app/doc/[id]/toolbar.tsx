'use client';

import type { EditorView } from '@codemirror/view';
import { CODE_SNIPPET, TABLE_SNIPPET, TASK_SNIPPET, prefixLines, wrapSelection } from '../../../lib/format';

/**
 * Editor toolbar (F9-3). Executes pure text transforms from lib/format.ts as
 * CodeMirror transactions. Operates on the live view via getView — Yjs sync
 * picks the change up through y-codemirror.next, so no CRDT code here.
 */
export function Toolbar({ getView }: { getView: () => EditorView | null }) {
  function transact(kind: 'wrap' | 'prefix' | 'insert', a: string, b?: string) {
    const view = getView();
    if (!view) return;
    const sel = view.state.selection.main;
    const doc = view.state.doc.toString();
    if (kind === 'wrap') {
      const r = wrapSelection(doc, sel.from, sel.to, a, b ?? '');
      view.dispatch({
        changes: { from: 0, to: doc.length, insert: r.text },
        selection: { anchor: r.start, head: r.end },
      });
    } else if (kind === 'prefix') {
      const r = prefixLines(doc, sel.from, sel.to, a);
      view.dispatch({
        changes: { from: 0, to: doc.length, insert: r.text },
        selection: { anchor: r.start, head: r.end },
      });
    } else {
      view.dispatch({
        changes: { from: sel.from, to: sel.to, insert: a },
        selection: { anchor: sel.from + a.length },
      });
    }
    view.focus();
  }

  const btn: React.CSSProperties = {
    border: '1px solid #ddd',
    borderRadius: 6,
    background: '#fff',
    padding: '4px 10px',
    fontSize: 13,
    cursor: 'pointer',
  };

  return (
    <div data-testid="toolbar" style={{ display: 'flex', gap: 6, flexWrap: 'wrap', margin: '8px 0' }}>
      <button style={btn} data-testid="toolbar-bold" title="Bold" onClick={() => transact('wrap', '**', '**')}>
        B
      </button>
      <button style={btn} data-testid="toolbar-italic" title="Italic" onClick={() => transact('wrap', '*', '*')}>
        I
      </button>
      <button style={btn} data-testid="toolbar-h1" title="Heading 1" onClick={() => transact('prefix', '# ')}>
        H1
      </button>
      <button style={btn} data-testid="toolbar-h2" title="Heading 2" onClick={() => transact('prefix', '## ')}>
        H2
      </button>
      <button style={btn} data-testid="toolbar-list" title="Bullet list" onClick={() => transact('prefix', '- ')}>
        List
      </button>
      <button style={btn} data-testid="toolbar-task" title="Task list" onClick={() => transact('insert', TASK_SNIPPET)}>
        Task
      </button>
      <button style={btn} data-testid="toolbar-table" title="Table" onClick={() => transact('insert', TABLE_SNIPPET)}>
        Table
      </button>
      <button
        style={btn}
        data-testid="toolbar-code"
        title="Code block"
        onClick={() => transact('wrap', CODE_SNIPPET.before, CODE_SNIPPET.after)}
      >
        Code
      </button>
    </div>
  );
}
