'use server';

import { redirect } from 'next/navigation';
import { revalidatePath } from 'next/cache';
import { createDocumentSchema, emailSchema } from 'shared';
import {
  ShareError,
  createDocument,
  deleteDocument,
  getOwnedDocument,
  renameDocument,
  shareDocument,
  unshareDocument,
} from '../../lib/document-service';
import { RestoreError, restoreAt } from '../../lib/restore-service';
import { requireSession } from '../../lib/session';

export interface CreateDocState {
  error?: string;
}

export async function createDocumentAction(
  _prev: CreateDocState,
  formData: FormData,
): Promise<CreateDocState> {
  const user = await requireSession();
  const parsed = createDocumentSchema.safeParse({ title: formData.get('title') });
  if (!parsed.success) return { error: 'Title must be 1–200 characters.' };

  const doc = await createDocument(user.id, parsed.data.title);
  redirect(`/doc/${doc.id}`);
}

export interface ShareState {
  error?: string;
  ok?: string;
}

/** Owner-only: grant another user (by email) collaborator access with a role (Faz 8, roles F12). */
export async function shareDocumentAction(
  docId: string,
  _prev: ShareState,
  formData: FormData,
): Promise<ShareState> {
  const user = await requireSession();
  if (!(await getOwnedDocument(docId, user.id))) return { error: 'Not found.' };

  const parsed = emailSchema.safeParse(formData.get('email'));
  if (!parsed.success) return { error: 'Enter a valid email.' };
  const role = formData.get('role');
  if (role !== 'editor' && role !== 'viewer') return { error: 'Pick a role.' };

  // Normalize: trim and lower-case before lookup — matches auth registration (service.go Trim+ToLower).
  const email = parsed.data.trim().toLowerCase();

  try {
    const added = await shareDocument(docId, user.id, email, role);
    revalidatePath(`/doc/${docId}`);
    return { ok: `Shared with ${added.email} (${added.role}).` };
  } catch (err) {
    if (err instanceof ShareError) return { error: err.message };
    throw err;
  }
}

/** Owner-only: revoke a collaborator's access (Faz 8). */
export async function unshareDocumentAction(docId: string, userId: string): Promise<void> {
  const user = await requireSession();
  if (!(await getOwnedDocument(docId, user.id))) return;
  await unshareDocument(docId, userId);
  revalidatePath(`/doc/${docId}`);
}

export interface RenameState {
  error?: string;
  ok?: string;
}

/** Owner-only: rename a document (Faz 8). */
export async function renameDocumentAction(
  docId: string,
  _prev: RenameState,
  formData: FormData,
): Promise<RenameState> {
  const user = await requireSession();
  const parsed = createDocumentSchema.safeParse({ title: formData.get('title') });
  if (!parsed.success) return { error: 'Title must be 1–200 characters.' };

  const ok = await renameDocument(docId, user.id, parsed.data.title);
  if (!ok) return { error: 'Not found.' };
  revalidatePath(`/doc/${docId}`);
  revalidatePath('/documents');
  return { ok: 'Renamed.' };
}

/** Owner-only: delete a document and its history (Faz 8), then back to the list. */
export async function deleteDocumentAction(docId: string): Promise<void> {
  const user = await requireSession();
  await deleteDocument(docId, user.id); // no-op if not owned
  redirect('/documents');
}

export interface RestoreState {
  error?: string;
  ok?: string;
}

/** Restore the document text to a past replay point (F11). Accessible editors only. */
export async function restoreDocumentAction(docId: string, at: string): Promise<RestoreState> {
  try {
    await restoreAt(docId, at);
    revalidatePath(`/doc/${docId}`);
    revalidatePath(`/doc/${docId}/history`);
    return { ok: `Restored to update #${at}.` };
  } catch (err) {
    if (err instanceof RestoreError) return { error: err.message };
    throw err;
  }
}
