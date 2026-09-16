import { describe, expect, it } from 'vitest';
import { signSession, verifySession } from 'protocol';

/**
 * The JWT layer shared by web (cookie) and the Go sync server (handshake). getSession /
 * requireSession wrap these; the cookie plumbing itself is exercised via the app.
 */
describe('session JWT (protocol)', () => {
  it('round-trips a session user', async () => {
    const token = await signSession({ id: 'user_123', email: 'a@b.com' });
    const user = await verifySession(token);
    expect(user).toEqual({ id: 'user_123', email: 'a@b.com' });
  });

  it('returns null for a tampered token', async () => {
    const token = await signSession({ id: 'user_123', email: 'a@b.com' });
    const tampered = `${token.slice(0, -2)}xy`;
    expect(await verifySession(tampered)).toBeNull();
  });

  it('returns null for garbage', async () => {
    expect(await verifySession('not.a.jwt')).toBeNull();
  });
});
