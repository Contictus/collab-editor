import { SignJWT, jwtVerify } from 'jose';
import type { SessionUser } from 'shared';

/**
 * Shared auth + WS protocol surface. Both web (REST/Server Actions) and
 * ws-server (handshake) sign/verify the SAME JWT, so identity is unified.
 * Faz 0 provides the token helpers + message-type placeholders; the sync
 * message wire format itself is y-protocols binary (Faz 3).
 */

const COOKIE_NAME = 'session';
export const SESSION_COOKIE = COOKIE_NAME;

function secretKey(): Uint8Array {
  const secret = process.env.JWT_SECRET;
  if (!secret) throw new Error('JWT_SECRET is not set (set it in root .env, see .env.example)');
  return new TextEncoder().encode(secret);
}

export async function signSession(user: SessionUser): Promise<string> {
  return new SignJWT({ email: user.email })
    .setProtectedHeader({ alg: 'HS256' })
    .setSubject(user.id)
    .setIssuedAt()
    .setExpirationTime('7d')
    .sign(secretKey());
}

export async function verifySession(token: string): Promise<SessionUser | null> {
  try {
    const { payload } = await jwtVerify(token, secretKey());
    if (typeof payload.sub !== 'string' || typeof payload.email !== 'string') return null;
    return { id: payload.sub, email: payload.email };
  } catch {
    return null;
  }
}

/**
 * Client→server / server→client WS message discriminants (y-protocols).
 * Only Sync and Awareness are used — Auth is handled at the HTTP upgrade (INVARIANT #2),
 * not as a WS message, so no MessageType.Auth exists.
 */
export const MessageType = {
  Sync: 0,
  Awareness: 1,
} as const;
export type MessageType = (typeof MessageType)[keyof typeof MessageType];
