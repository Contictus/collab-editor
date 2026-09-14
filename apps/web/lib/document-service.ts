import { prisma } from 'db';
import { loadText } from 'shared/crdt';

/**
 * Document domain logic (Faz 2). Every read/write is owner-scoped — a user only
 * sees and loads their own documents. Live editing (WS) is Faz 3+; here we do
 * CRUD + the read-only SSR text bootstrap.
 */

export interface DocumentSummary {
  id: string;
  title: string;
  updatedAt: Date;
  /** false when the document is shared with the user by another owner (Faz 8). */
  isOwner: boolean;
}

export async function createDocument(ownerId: string, title: string): Promise<{ id: string }> {
  const doc = await prisma.document.create({ data: { title, ownerId }, select: { id: true } });
  return doc;
}

/** Documents the user can open: their own plus those shared with them (Faz 8). */
export async function listDocuments(userId: string): Promise<DocumentSummary[]> {
  const docs = await prisma.document.findMany({
    where: { OR: [{ ownerId: userId }, { collaborators: { some: { userId } } }] },
    orderBy: { updatedAt: 'desc' },
    select: { id: true, title: true, updatedAt: true, ownerId: true },
  });
  return docs.map((d) => ({
    id: d.id,
    title: d.title,
    updatedAt: d.updatedAt,
    isOwner: d.ownerId === userId,
  }));
}

/**
 * Rename a document (owner-only, Faz 8). The ownerId is part of the WHERE, so a
 * non-owner's call matches zero rows — returns false rather than touching data.
 */
export async function renameDocument(id: string, ownerId: string, title: string): Promise<boolean> {
  const res = await prisma.document.updateMany({ where: { id, ownerId }, data: { title } });
  return res.count > 0;
}

/**
 * Delete a document (owner-only, Faz 8). Cascades to the op log, snapshots, and
 * collaborator grants (all onDelete: Cascade). Returns false if not owned.
 */
export async function deleteDocument(id: string, ownerId: string): Promise<boolean> {
  const res = await prisma.document.deleteMany({ where: { id, ownerId } });
  return res.count > 0;
}

/** Returns the document only if it belongs to ownerId, else null. Owner-only ops. Used by rename/delete guards. */
export function getOwnedDocument(id: string, ownerId: string) {
  return prisma.document.findFirst({
    where: { id, ownerId },
    select: { id: true, title: true },
  });
}

/**
 * Access gate (Faz 8): the document if the user is the owner OR a collaborator,
 * else null (404/403 upstream). This is the SAME predicate the ws-server enforces
 * at the handshake — REST reads and live sync share one authorization model.
 */
export async function getAccessibleDocument(id: string, userId: string) {
  const doc = await prisma.document.findFirst({
    where: { id, OR: [{ ownerId: userId }, { collaborators: { some: { userId } } }] },
    select: { id: true, title: true, ownerId: true },
  });
  if (!doc) return null;
  return { id: doc.id, title: doc.title, isOwner: doc.ownerId === userId };
}

export class ShareError extends Error {}

export interface CollaboratorSummary {
  userId: string;
  email: string;
  createdAt: Date;
}

/**
 * Grant a user (by email) collaborator access to a document (Faz 8). Owner-only:
 * the caller must pass the verified ownerId. Idempotent — re-sharing is a no-op.
 * Throws ShareError on unknown email, self-share, or a non-owned document.
 */
export async function shareDocument(
  documentId: string,
  ownerId: string,
  inviteeEmail: string,
): Promise<CollaboratorSummary> {
  const doc = await prisma.document.findFirst({
    where: { id: documentId, ownerId },
    select: { id: true },
  });
  if (!doc) throw new ShareError('Document not found.');

  const invitee = await prisma.user.findUnique({
    where: { email: inviteeEmail },
    select: { id: true, email: true },
  });
  if (!invitee) throw new ShareError('No user with that email.');
  if (invitee.id === ownerId) throw new ShareError('You already own this document.');

  const row = await prisma.documentCollaborator.upsert({
    where: { documentId_userId: { documentId, userId: invitee.id } },
    create: { documentId, userId: invitee.id },
    update: {},
    select: { createdAt: true },
  });
  return { userId: invitee.id, email: invitee.email, createdAt: row.createdAt };
}

/** Collaborators of a document (owner-scoped; caller verifies ownership). */
export async function listCollaborators(documentId: string): Promise<CollaboratorSummary[]> {
  const rows = await prisma.documentCollaborator.findMany({
    where: { documentId },
    orderBy: { createdAt: 'asc' },
    select: { userId: true, createdAt: true, user: { select: { email: true } } },
  });
  return rows.map((r) => ({ userId: r.userId, email: r.user.email, createdAt: r.createdAt }));
}

/** Revoke a collaborator's access (owner-only; caller verifies ownership). */
export async function unshareDocument(documentId: string, userId: string): Promise<void> {
  await prisma.documentCollaborator.deleteMany({ where: { documentId, userId } });
}

/**
 * Read-only SSR bootstrap text: latest snapshot + updates recorded after it,
 * replayed via the shared CRDT helper (load-on-open, data-model.md). Empty string
 * until sync/persistence phases populate the op log.
 */
export async function loadInitialText(documentId: string): Promise<string> {
  const snapshot = await prisma.documentSnapshot.findFirst({
    where: { documentId },
    orderBy: { id: 'desc' },
    select: { state: true, upToUpdateId: true },
  });

  const updates = await prisma.documentUpdate.findMany({
    where: { documentId, ...(snapshot ? { id: { gt: snapshot.upToUpdateId } } : {}) },
    orderBy: { id: 'asc' },
    select: { update: true },
  });

  return loadText(
    snapshot ? new Uint8Array(snapshot.state) : null,
    updates.map((u) => new Uint8Array(u.update)),
  );
}
