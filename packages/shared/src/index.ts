import { z } from 'zod';

/**
 * Shared types and zod schemas used by the web app (UI, actions, API contracts).
 * Faz 0: auth + document input shapes. Extended in later phases.
 */

export const emailSchema = z.string().email();
export const passwordSchema = z.string().min(8).max(200);

export const credentialsSchema = z.object({
  email: emailSchema,
  password: passwordSchema,
});
export type Credentials = z.infer<typeof credentialsSchema>;

export const createDocumentSchema = z.object({
  title: z.string().min(1).max(200),
});
export type CreateDocumentInput = z.infer<typeof createDocumentSchema>;

/** Authenticated user identity carried in the JWT (Faz 1). */
export interface SessionUser {
  id: string;
  email: string;
}

/** Compaction threshold: snapshot after this many appended updates (data-model.md). */
export const SNAPSHOT_THRESHOLD = 100;
