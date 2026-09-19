package db

import (
	"context"
)

// Roles and public links (F12). Role is 'editor' (read+write) or 'viewer'
// (read-only); unknown values are rejected before they reach SQL.

// validRole reports whether role is a known collaborator role.
func validRole(role string) bool {
	return role == "editor" || role == "viewer"
}

// ShareDocumentWithRole grants inviteeID access with a role. Idempotent on the
// grant, but re-sharing upgrades/downgrades the role to the requested one.
func (s *Store) ShareDocumentWithRole(ctx context.Context, documentID, ownerID, inviteeID, role string) (*Collaborator, error) {
	if !validRole(role) {
		return nil, ErrBadRole
	}
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
		`INSERT INTO "DocumentCollaborator" ("documentId", "userId", "role") VALUES ($1, $2, $3)
		 ON CONFLICT ("documentId", "userId") DO UPDATE SET "role" = $3`, documentID, inviteeID, role)
	if err != nil {
		return nil, err
	}
	var c Collaborator
	var email string
	err = s.pool.QueryRow(ctx,
		`SELECT c."userId", u."email", c."role", c."createdAt" FROM "DocumentCollaborator" c
		 JOIN "User" u ON u."id" = c."userId"
		 WHERE c."documentId" = $1 AND c."userId" = $2`, documentID, inviteeID,
	).Scan(&c.UserID, &email, &c.Role, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	c.Email = email
	return &c, nil
}

// CheckDocumentRole reports the access level: 'owner', 'editor', 'viewer',
// or '' when the user has no access.
func (s *Store) CheckDocumentRole(ctx context.Context, docID, userID string) (string, error) {
	var ownerID string
	if err := s.pool.QueryRow(ctx,
		`SELECT "ownerId" FROM "Document" WHERE "id" = $1`, docID,
	).Scan(&ownerID); err != nil {
		if isNoRows(err) {
			return "", nil
		}
		return "", err
	}
	if ownerID == userID {
		return "owner", nil
	}
	var role string
	if err := s.pool.QueryRow(ctx,
		`SELECT "role" FROM "DocumentCollaborator" WHERE "documentId" = $1 AND "userId" = $2`,
		docID, userID,
	).Scan(&role); err != nil {
		if isNoRows(err) {
			return "", nil
		}
		return "", err
	}
	return role, nil
}

// SetPublicID enables (random token) or disables (empty) the public
// read-only link. Owner-only: matches zero rows otherwise.
func (s *Store) SetPublicID(ctx context.Context, documentID, ownerID string, enable bool) (*string, error) {
	var publicID *string
	if enable {
		token, err := newID()
		if err != nil {
			return nil, err
		}
		publicID = &token
	}
	tag, err := s.pool.Exec(ctx,
		`UPDATE "Document" SET "publicId" = $3, "updatedAt" = NOW()
		 WHERE "id" = $1 AND "ownerId" = $2`, documentID, ownerID, publicID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotOwned
	}
	return publicID, nil
}

// GetByPublicID returns the document for a public token, else nil. No auth —
// the token IS the capability (unlisted, random 25 chars).
func (s *Store) GetByPublicID(ctx context.Context, publicID string) (*Document, error) {
	var d Document
	err := s.pool.QueryRow(ctx,
		`SELECT "id", "title", "ownerId", "publicId", "createdAt", "updatedAt" FROM "Document"
		 WHERE "publicId" = $1`, publicID,
	).Scan(&d.ID, &d.Title, &d.OwnerID, &d.PublicID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}
