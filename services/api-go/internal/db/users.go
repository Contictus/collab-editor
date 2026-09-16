package db

import (
	"context"
)

// Users mirrors auth-service.ts: register (unique email) + login (by email).

// FindUserByEmail returns the user row for login/register checks, or nil when unknown.
func (s *Store) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT "id", "email", "password", "createdAt" FROM "User" WHERE "email" = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.CreatedAt)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// FindUserByID returns the user row, or nil when unknown.
func (s *Store) FindUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT "id", "email", "password", "createdAt" FROM "User" WHERE "id" = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Password, &u.CreatedAt)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// DeleteUser removes a user row (test cleanup; documents must go first —
// the owner FK is RESTRICT, mirroring Prisma).
func (s *Store) DeleteUser(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM "User" WHERE "id" = $1`, id)
	return err
}

// CreateUser inserts a user. passwordHash must already be an argon2id PHC string
// (hashing lives in the auth layer, F2) — the store never sees plaintext.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (*User, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}
	var u User
	err = s.pool.QueryRow(ctx,
		`INSERT INTO "User" ("id", "email", "password") VALUES ($1, $2, $3)
		 RETURNING "id", "email", "password", "createdAt"`,
		id, email, passwordHash,
	).Scan(&u.ID, &u.Email, &u.Password, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
