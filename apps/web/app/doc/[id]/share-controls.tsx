'use client';

import { useActionState } from 'react';
import {
  shareDocumentAction,
  unshareDocumentAction,
  type ShareState,
} from '../../actions/documents';
import type { CollaboratorSummary } from '../../../lib/document-service';

const initial: ShareState = {};

/**
 * Owner-only sharing panel (Faz 8): invite a user by email to co-edit, and revoke
 * access. Rendered only for the document owner. Collaborators gain the same live
 * WS + REST access the owner has (owner-OR-collaborator predicate).
 */
export function ShareControls({
  docId,
  collaborators,
}: {
  docId: string;
  collaborators: CollaboratorSummary[];
}) {
  const action = shareDocumentAction.bind(null, docId);
  const [state, formAction, pending] = useActionState(action, initial);

  return (
    <section style={{ display: 'grid', gap: 8, maxWidth: 420, margin: '8px 0' }}>
      <form action={formAction} style={{ display: 'flex', gap: 8 }}>
        <input
          name="email"
          type="email"
          required
          placeholder="Invite by email"
          data-testid="share-email"
          style={{ flex: 1 }}
        />
        <button type="submit" disabled={pending}>
          {pending ? 'Sharing…' : 'Share'}
        </button>
      </form>
      {state.error && <p style={{ color: 'crimson', margin: 0 }}>{state.error}</p>}
      {state.ok && <p style={{ color: '#22a565', margin: 0 }}>{state.ok}</p>}

      {collaborators.length > 0 && (
        <ul style={{ margin: 0, paddingLeft: 18, color: '#555' }}>
          {collaborators.map((c) => (
            <li key={c.userId} style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <span>{c.email}</span>
              <form action={unshareDocumentAction.bind(null, docId, c.userId)}>
                <button type="submit" style={{ fontSize: 12 }}>
                  remove
                </button>
              </form>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
