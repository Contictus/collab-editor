'use client';

import { useActionState } from 'react';
import { loginAction, type AuthFormState } from '../actions/auth';

const initial: AuthFormState = {};

export function LoginForm() {
  const [state, action, pending] = useActionState(loginAction, initial);
  return (
    <form action={action} style={{ display: 'grid', gap: 12, maxWidth: 320 }}>
      <label>
        Email
        <input name="email" type="email" required autoComplete="email" />
      </label>
      <label>
        Password
        <input name="password" type="password" required autoComplete="current-password" />
      </label>
      {state.error && <p style={{ color: 'crimson', margin: 0 }}>{state.error}</p>}
      <button type="submit" disabled={pending}>
        {pending ? 'Signing in…' : 'Sign in'}
      </button>
    </form>
  );
}
