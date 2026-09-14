'use client';

import { useActionState } from 'react';
import { createDocumentAction, type CreateDocState } from '../actions/documents';

const initial: CreateDocState = {};

export function CreateDocumentForm() {
  const [state, action, pending] = useActionState(createDocumentAction, initial);
  return (
    <form action={action} style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
      <input
        name="title"
        placeholder="New document title"
        required
        maxLength={200}
        style={{ flex: 1, minWidth: 180, padding: '8px 10px', border: '1px solid #ddd', borderRadius: 8, background: '#fff' }}
      />
      <button
        type="submit"
        disabled={pending}
        style={{
          padding: '8px 14px',
          borderRadius: 999,
          border: '1px solid #111',
          background: '#111',
          color: '#fff',
          fontWeight: 600,
          fontSize: 13,
          cursor: pending ? 'wait' : 'pointer',
        }}
      >
        {pending ? 'Creating…' : 'Create'}
      </button>
      {state.error && <span style={{ color: 'crimson', fontSize: 13 }}>{state.error}</span>}
    </form>
  );
}
