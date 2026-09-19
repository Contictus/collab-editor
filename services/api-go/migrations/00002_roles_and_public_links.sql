-- +goose Up
-- Read-only collaborators + public read-only links (F12, mirrors Prisma
-- 20260919120000_collaborator_role_and_public_link). IF NOT EXISTS / guarded
-- adds so Prisma-managed databases apply it as a harmless no-op.

ALTER TABLE "Document" ADD COLUMN IF NOT EXISTS "publicId" TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS "Document_publicId_key" ON "Document"("publicId");

ALTER TABLE "DocumentCollaborator" ADD COLUMN IF NOT EXISTS "role" TEXT NOT NULL DEFAULT 'editor';

-- +goose Down
-- Intentionally empty: role/publicId columns stay (data-preserving).
SELECT 1;
