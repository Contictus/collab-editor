-- +goose Up
-- Baseline mirroring the Prisma migrations (packages/db/prisma/migrations).
-- Same tables, same quoted identifiers, same constraints — Prisma and Go read
-- and write the same rows. IF NOT EXISTS so applying on a Prisma-migrated
-- database is a harmless no-op, while a fresh database is bootstrapped.
-- Prisma remains the migration owner for now; this baseline only guarantees
-- the Go service can start against an empty database.

CREATE TABLE IF NOT EXISTS "User" (
    "id" TEXT NOT NULL,
    "email" TEXT NOT NULL,
    "password" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "User_pkey" PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "Document" (
    "id" TEXT NOT NULL,
    "title" TEXT NOT NULL,
    "ownerId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    CONSTRAINT "Document_pkey" PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "DocumentUpdate" (
    "id" BIGSERIAL NOT NULL,
    "documentId" TEXT NOT NULL,
    "update" BYTEA NOT NULL,
    "clock" INTEGER NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "DocumentUpdate_pkey" PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "DocumentSnapshot" (
    "id" BIGSERIAL NOT NULL,
    "documentId" TEXT NOT NULL,
    "state" BYTEA NOT NULL,
    "upToUpdateId" BIGINT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "DocumentSnapshot_pkey" PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "DocumentCollaborator" (
    "documentId" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "DocumentCollaborator_pkey" PRIMARY KEY ("documentId","userId")
);

CREATE UNIQUE INDEX IF NOT EXISTS "User_email_key" ON "User"("email");
CREATE INDEX IF NOT EXISTS "DocumentUpdate_documentId_id_idx" ON "DocumentUpdate"("documentId", "id");
CREATE INDEX IF NOT EXISTS "DocumentSnapshot_documentId_id_idx" ON "DocumentSnapshot"("documentId", "id");
CREATE INDEX IF NOT EXISTS "DocumentCollaborator_userId_idx" ON "DocumentCollaborator"("userId");

-- NOTE: foreign keys are NOT created here. Plain ALTER TABLE ... ADD CONSTRAINT
-- is not idempotent and goose splits statements, so conditional DDL cannot live
-- in this file. Constraints are ensured by db.EnsureConstraints (Go code, ignores
-- duplicate_object), which the migrate subcommand runs after goose up. Prisma
-- remains the migration owner; this baseline only guarantees the tables exist.

-- +goose Down
-- Intentionally empty: the baseline must never drop Prisma-owned tables.
SELECT 1;
