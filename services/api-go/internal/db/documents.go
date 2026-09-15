package db

import (
	"context"
)

// Documents mirrors document-service.ts predicate-for-predicate:
// owner-scoped writes, owner-OR-collaborator reads, idempotent share.

// accessPredicate is the shared authorization model: the document when the user
// is the owner OR a collaborator, else no row. Same predicate the ws-server
// handshake enforces (authorizeDocument) — REST and live sync share it.
const accessPredicate = `"id" = $1 AND ("ownerId" = $2 OR EXISTS (
	SELECT 1 FROM "DocumentCollaborator" c
	WHERE c."documentId" = "Document"."id" AND c."userId" = $2
))`

// CreateDocument inserts a document owned by ownerID.
func (s *Store) CreateDocument(ctx context.Context, ownerID, title string) (*Document, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}
	var d Document
	err = s.pool.QueryRow(ctx,
		`INSERT INTO "Document" ("id", "title", "ownerId", "updatedAt")
		 VALUES ($1, $2, $3, NOW())
		 RETURNING "id", "title", "ownerId", "createdAt", "updatedAt"`,
		id, title, ownerID,
	).Scan(&d.ID, &d.Title, &d.OwnerID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// ListDocuments returns openable documents: owned plus shared (Faz 8), newest first.
func (s *Store) ListDocuments(ctx context.Context, userID string) ([]DocumentSummary, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT "id", "title", "ownerId", "updatedAt" FROM "Document"
		 WHERE "ownerId" = $1 OR EXISTS (
			SELECT 1 FROM "DocumentCollaborator" c
			WHERE c."documentId" = "Document"."id" AND c."userId" = $1
		 )
		 ORDER BY "updatedAt" DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DocumentSummary
	for rows.Next() {
		var d DocumentSummary
		var ownerID string
		if err := rows.Scan(&d.ID, &d.Title, &ownerID, &d.UpdatedAt); err != nil {
			return nil, err
		}
		d.IsOwner = ownerID == userID
		out = append(out, d)
	}
	return out, rows.Err()
}

// RenameDocument renames an owned document. updatedAt is bumped explicitly:
// Prisma's @updatedAt is client-side, raw SQL must set it itself.
// Returns false when the document is not owned by ownerID.
func (s *Store) RenameDocument(ctx context.Context, id, ownerID, title string) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE "Document" SET "title" = $3, "updatedAt" = NOW()
		 WHERE "id" = $1 AND "ownerId" = $2`, id, ownerID, title)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// DeleteDocument deletes an owned document (op log, snapshots, grants cascade).
// Returns false when the document is not owned by ownerID.
func (s *Store) DeleteDocument(ctx context.Context, id, ownerID string) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM "Document" WHERE "id" = $1 AND "ownerId" = $2`, id, ownerID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// GetAccessibleDocument returns id/title/isOwner when the user is the owner OR a
// collaborator, else nil. Same predicate as the WS handshake.
func (s *Store) GetAccessibleDocument(ctx context.Context, id, userID string) (*DocumentSummary, error) {
	var d DocumentSummary
	var ownerID string
	err := s.pool.QueryRow(ctx,
		`SELECT "id", "title", "ownerId" FROM "Document" WHERE `+accessPredicate, id, userID,
	).Scan(&d.ID, &d.Title, &ownerID)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	d.IsOwner = ownerID == userID
	return &d, nil
}

// CheckDocumentAccess reports whether userID may open (and sync) the document:
// owner OR collaborator. This is authorizeDocument in ws-server/src/auth.ts.
func (s *Store) CheckDocumentAccess(ctx context.Context, docID, userID string) (bool, error) {
	var one int
	err := s.pool.QueryRow(ctx,
		`SELECT 1 FROM "Document" WHERE `+accessPredicate, docID, userID,
	).Scan(&one)
	if err != nil {
		if isNoRows(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ShareDocument grants inviteeID access to an owned document. Idempotent:
// re-sharing is a no-op returning the existing grant.
func (s *Store) ShareDocument(ctx context.Context, documentID, ownerID, inviteeID string) (*Collaborator, error) {
	var owned string
	if err := s.pool.QueryRow(ctx,
		`SELECT "id" FROM "Document" WHERE "id" = $1 AND "ownerId" = $2`, documentID, ownerID,
	).Scan(&owned); err != nil {
		if isNoRows(err) {
			return nil, ErrNotOwned
		}
		return nil, err
	}
	if inviteeID == ownerID {
		return nil, ErrSelfShare
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO "DocumentCollaborator" ("documentId", "userId") VALUES ($1, $2)
		 ON CONFLICT ("documentId", "userId") DO NOTHING`, documentID, inviteeID)
	if err != nil {
		return nil, err
	}
	var c Collaborator
	var email string
	err = s.pool.QueryRow(ctx,
		`SELECT c."userId", u."email", c."createdAt" FROM "DocumentCollaborator" c
		 JOIN "User" u ON u."id" = c."userId"
		 WHERE c."documentId" = $1 AND c."userId" = $2`, documentID, inviteeID,
	).Scan(&c.UserID, &email, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	c.Email = email
	return &c, nil
}

// ListCollaborators returns the grants on a document, oldest first.
func (s *Store) ListCollaborators(ctx context.Context, documentID string) ([]Collaborator, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT c."userId", u."email", c."createdAt" FROM "DocumentCollaborator" c
		 JOIN "User" u ON u."id" = c."userId"
		 WHERE c."documentId" = $1 ORDER BY c."createdAt" ASC`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Collaborator
	for rows.Next() {
		var c Collaborator
		if err := rows.Scan(&c.UserID, &c.Email, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UnshareDocument revokes a grant.
func (s *Store) UnshareDocument(ctx context.Context, documentID, userID string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM "DocumentCollaborator" WHERE "documentId" = $1 AND "userId" = $2`,
		documentID, userID)
	return err
}
