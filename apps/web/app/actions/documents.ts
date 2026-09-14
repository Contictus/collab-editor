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

/** Owner-only: grant another user (by email) collaborator access (Faz 8). */
export async function shareDocumentAction(
  docId: string,
  _prev: ShareState,
  formData: FormData,
): Promise<ShareState> {
  const user = await requireSession();
  if (!(await getOwnedDocument(docId, user.id))) return { error: 'Not found.' };

  const parsed = emailSchema.safeParse(formData.get('email'));
  if (!parsed.success) return { error: 'Enter a valid email.' };

  try {
    const added = await shareDocument(docId, user.id, parsed.data);
    revalidatePath(`/doc/${docId}`);
    return { ok: `Shared with ${added.email}.` };
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
