import { hash, verify } from '@node-rs/argon2';

/**
 * Password hashing (Faz 1). @node-rs/argon2 defaults to the argon2id variant,
 * which is what the data model requires. Ships prebuilt binaries (no compile on
 * Windows). Pure functions — unit-tested without a DB.
 */
export function hashPassword(plain: string): Promise<string> {
  return hash(plain);
}

export async function verifyPassword(hashed: string, plain: string): Promise<boolean> {
  try {
    return await verify(hashed, plain);
  } catch {
    // Malformed/invalid hash string → treat as non-match rather than throwing.
    return false;
  }
}
