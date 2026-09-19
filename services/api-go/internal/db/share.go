package db

import "errors"

// Share errors mirror ShareError cases in document-service.ts.
var (
	// ErrNotOwned is returned when sharing a document the caller does not own.
	ErrNotOwned = errors.New("db: document not found or not owned")
	// ErrSelfShare is returned when sharing with the owner.
	ErrSelfShare = errors.New("db: cannot share with the owner")
	// ErrBadRole is returned for unknown collaborator roles.
	ErrBadRole = errors.New("db: unknown role (want editor or viewer)")
)
