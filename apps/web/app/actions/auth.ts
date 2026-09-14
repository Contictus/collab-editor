'use server';

import { headers } from 'next/headers';
import { redirect } from 'next/navigation';
import { credentialsSchema } from 'shared';
import { EmailTakenError, loginUser, registerUser } from '../../lib/auth-service';
import { rateLimit } from '../../lib/rate-limit';
import { clearSession, createSession } from '../../lib/session';

export interface AuthFormState {
  error?: string;
}

// Brute-force / credential-stuffing throttles (Faz 8 hardening).
const LOGIN_PER_IP = { limit: 10, windowMs: 60_000 };
const LOGIN_PER_EMAIL = { limit: 5, windowMs: 60_000 };
const REGISTER_PER_IP = { limit: 5, windowMs: 600_000 };

function parse(formData: FormData) {
  return credentialsSchema.safeParse({
    email: formData.get('email'),
    password: formData.get('password'),
  });
}

/** Best-effort client IP from proxy headers; 'local' in unproxied dev. */
async function clientIp(): Promise<string> {
  const h = await headers();
  const fwd = h.get('x-forwarded-for');
  if (fwd) return fwd.split(',')[0]!.trim();
  return h.get('x-real-ip') ?? 'local';
}

export async function registerAction(
  _prev: AuthFormState,
  formData: FormData,
): Promise<AuthFormState> {
  const ip = await clientIp();
  if (!rateLimit(`register:ip:${ip}`, REGISTER_PER_IP.limit, REGISTER_PER_IP.windowMs).ok) {
    return { error: 'Too many attempts. Please try again later.' };
  }

  const parsed = parse(formData);
  if (!parsed.success) {
    return { error: 'Enter a valid email and a password of at least 8 characters.' };
  }
  try {
    const user = await registerUser(parsed.data);
    await createSession(user);
  } catch (err) {
    if (err instanceof EmailTakenError) return { error: 'That email is already registered.' };
    throw err;
  }
  redirect('/documents');
}

export async function loginAction(
  _prev: AuthFormState,
  formData: FormData,
): Promise<AuthFormState> {
  const ip = await clientIp();
  if (!rateLimit(`login:ip:${ip}`, LOGIN_PER_IP.limit, LOGIN_PER_IP.windowMs).ok) {
    return { error: 'Too many attempts. Please try again shortly.' };
  }

  const parsed = parse(formData);
  if (!parsed.success) return { error: 'Invalid email or password.' };

  // Per-email throttle: caps guessing against one account regardless of source IP.
  if (!rateLimit(`login:email:${parsed.data.email}`, LOGIN_PER_EMAIL.limit, LOGIN_PER_EMAIL.windowMs).ok) {
    return { error: 'Too many attempts. Please try again shortly.' };
  }

  const user = await loginUser(parsed.data);
  if (!user) return { error: 'Invalid email or password.' };

  await createSession(user);
  redirect('/documents');
}

export async function logoutAction(): Promise<void> {
  await clearSession();
  redirect('/login');
}
