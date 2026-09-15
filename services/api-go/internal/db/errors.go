package db

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// isDuplicateObject reports Postgres 42710 (e.g. constraint already exists).
func isDuplicateObject(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42710"
}
