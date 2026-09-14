'use client';

import { useActionState } from 'react';
import { createDocumentAction, type CreateDocState } from '../actions/documents';

const initial: CreateDocState = {};

export function CreateDocumentForm() {
  const [state, action, pending] = useActionState(createDocumentAction, initial);
  return (
    <form action={action} style={{ display: 'flex', gap: 8, alignItems: 'center', margin: '16px 0' }}>
      <input name="title" placeholder="New document title" required maxLength={200} />
      <button type="submit" disabled={pending}>
        {pending ? 'Creating…' : 'Create'}
      </button>
      {state.error && <span style={{ color: 'crimson' }}>{state.error}</span>}
    </form>
  );
}
