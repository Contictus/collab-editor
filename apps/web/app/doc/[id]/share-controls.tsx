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
    <section
      style={{
        display: 'grid',
        gap: 10,
        maxWidth: 480,
        margin: '10px 0',
        padding: 12,
        border: '1px solid #e8e8e8',
        borderRadius: 8,
        background: '#fafaf7',
      }}
    >
      <div style={{ fontSize: 12, letterSpacing: 0.3, textTransform: 'uppercase', color: '#888' }}>
        Share — owner only
      </div>
      <form action={formAction} style={{ display: 'flex', gap: 8 }}>
        <input
          name="email"
          type="email"
          required
          placeholder="Invite by email"
          data-testid="share-email"
          style={{ flex: 1, padding: '6px 8px', border: '1px solid #ddd', borderRadius: 6 }}
        />
        <select
          name="role"
          defaultValue="editor"
          data-testid="share-role"
          title="Access level"
          style={{ padding: '6px 8px', border: '1px solid #ddd', borderRadius: 6, background: '#fff' }}
        >
          <option value="editor">Can edit</option>
          <option value="viewer">Can view</option>
        </select>
        <button
          type="submit"
          disabled={pending}
          style={{
            padding: '6px 12px',
            borderRadius: 6,
            border: '1px solid #111',
            background: '#111',
            color: '#fff',
            cursor: pending ? 'wait' : 'pointer',
          }}
        >
          {pending ? 'Sharing…' : 'Share'}
        </button>
      </form>
      {state.error && <p style={{ color: 'crimson', margin: 0, fontSize: 13 }}>{state.error}</p>}
      {state.ok && <p style={{ color: '#1a7a4a', margin: 0, fontSize: 13 }}>{state.ok}</p>}

      {collaborators.length > 0 ? (
        <ul style={{ margin: 0, padding: 0, listStyle: 'none', display: 'grid', gap: 6 }}>
          {collaborators.map((c) => (
            <li
              key={c.userId}
              style={{
                display: 'flex',
                gap: 8,
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '6px 8px',
                border: '1px solid #eee',
                borderRadius: 6,
                background: '#fff',
                fontSize: 13,
              }}
            >
              <span style={{ color: '#333' }}>
                {c.email}{' '}
                <span
                  data-testid={`collab-role-${c.userId}`}
                  title={c.role === 'viewer' ? 'Read-only' : 'Can edit'}
                  style={{
                    fontSize: 11,
                    padding: '1px 6px',
                    borderRadius: 4,
                    background: c.role === 'viewer' ? '#f0f0f0' : '#e6f4ea',
                    color: c.role === 'viewer' ? '#666' : '#1a7a4a',
                  }}
                >
                  {c.role}
                </span>
              </span>
              <form action={unshareDocumentAction.bind(null, docId, c.userId)}>
                <button
                  type="submit"
                  style={{ fontSize: 12, padding: '2px 8px', borderRadius: 4, border: '1px solid #ddd', background: '#fff' }}
                >
                  remove
                </button>
              </form>
            </li>
          ))}
        </ul>
      ) : (
        <p style={{ margin: 0, color: '#999', fontSize: 12 }}>No collaborators yet — invite by email above.</p>
      )}
    </section>
  );
}
