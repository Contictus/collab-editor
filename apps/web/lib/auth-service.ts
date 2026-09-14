import { prisma } from 'db';
import type { Credentials, SessionUser } from 'shared';
import { hashPassword, verifyPassword } from './password';

/**
 * Auth domain logic (Faz 1). No cookies/HTTP here — that lives in session.ts and
 * the Server Actions. Callers pass already-validated Credentials (zod at the edge).
 */

export class EmailTakenError extends Error {
  constructor() {
    super('Email already registered');
    this.name = 'EmailTakenError';
  }
}

export async function registerUser({ email, password }: Credentials): Promise<SessionUser> {
  const existing = await prisma.user.findUnique({ where: { email } });
  if (existing) throw new EmailTakenError();

  const user = await prisma.user.create({
    data: { email, password: await hashPassword(password) },
  });
  return { id: user.id, email: user.email };
}

/** Returns the user on valid credentials, or null on unknown email / wrong password. */
export async function loginUser({ email, password }: Credentials): Promise<SessionUser | null> {
  const user = await prisma.user.findUnique({ where: { email } });
  if (!user) return null;
  const ok = await verifyPassword(user.password, password);
  if (!ok) return null;
  return { id: user.id, email: user.email };
}
