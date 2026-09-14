'use client';

import { useActionState } from 'react';
import {
  deleteDocumentAction,
  renameDocumentAction,
  type RenameState,
} from '../../actions/documents';

const initial: RenameState = {};

/**
 * Owner-only document controls (Faz 8): rename inline and delete (with the whole
 * op-log history, via cascade). Rendered only for the owner. Delete asks for
 * client confirmation before the Server Action fires.
 */
export function DocControls({ docId, title }: { docId: string; title: string }) {
  const rename = renameDocumentAction.bind(null, docId);
  const [state, formAction, pending] = useActionState(rename, initial);
  const del = deleteDocumentAction.bind(null, docId);

  return (
    <div style={{ display: 'flex', gap: 12, alignItems: 'center', flexWrap: 'wrap', margin: '4px 0' }}>
      <form action={formAction} style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
        <input
          name="title"
          defaultValue={title}
          required
          maxLength={200}
          data-testid="rename-title"
          aria-label="Document title"
        />
        <button type="submit" disabled={pending}>
          {pending ? 'Saving…' : 'Rename'}
        </button>
      </form>

      <form
        action={del}
        onSubmit={(e) => {
          if (!confirm('Delete this document and its entire history? This cannot be undone.')) {
            e.preventDefault();
          }
        }}
      >
        <button type="submit" data-testid="delete-doc" style={{ color: '#c0392b' }}>
          Delete
        </button>
      </form>

      {state.error && <span style={{ color: 'crimson' }}>{state.error}</span>}
      {state.ok && <span style={{ color: '#22a565' }}>{state.ok}</span>}
    </div>
  );
}
