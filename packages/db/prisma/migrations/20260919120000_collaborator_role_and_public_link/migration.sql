-- Read-only collaborators + public read-only links (F12).
-- role: 'editor' (default, existing rows keep write) or 'viewer' (read-only).
-- publicId: nullable unique token for public read-only access; NULL means private.
ALTER TABLE "Document" ADD COLUMN "publicId" TEXT;

CREATE UNIQUE INDEX "Document_publicId_key" ON "Document"("publicId");

ALTER TABLE "DocumentCollaborator" ADD COLUMN "role" TEXT NOT NULL DEFAULT 'editor';
