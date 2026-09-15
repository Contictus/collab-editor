package db

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

// isNoRows reports pgx.ErrNoRows without importing it at every call site.
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
