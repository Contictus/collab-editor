'use client';

import { useActionState } from 'react';
import { registerAction, type AuthFormState } from '../actions/auth';

const initial: AuthFormState = {};

export function RegisterForm() {
  const [state, action, pending] = useActionState(registerAction, initial);
  return (
    <form action={action} style={{ display: 'grid', gap: 12, maxWidth: 320 }}>
      <label>
        Email
        <input name="email" type="email" required autoComplete="email" />
      </label>
      <label>
        Password (min 8 chars)
        <input name="password" type="password" required minLength={8} autoComplete="new-password" />
      </label>
      {state.error && <p style={{ color: 'crimson', margin: 0 }}>{state.error}</p>}
      <button type="submit" disabled={pending}>
        {pending ? 'Creating…' : 'Create account'}
      </button>
    </form>
  );
}
