// Package db is the Go persistence layer (F1).
//
// It talks to the SAME Postgres tables Prisma owns (packages/db/prisma/schema.prisma):
// same quoted identifiers, same columns, same access predicates as the Node
// services (document-service, auth-service, audit-service) and the ws-server
// persistence module. Prisma remains the migration owner; goose runs only the
// IF NOT EXISTS baseline so the Go service boots against an empty database too.
//
// Invariant parity: Compact inserts the snapshot BEFORE pruning covered updates
// in one transaction (invariant #3, snapshot commit → prune).
package db

import (
	"context"
	"database/sql"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/Contictus/collab-editor/services/api-go/migrations"
)

// prismaOnlyParams are DATABASE_URL query keys only Prisma understands. pgx
// would send them as server configuration and the connection dies with
// "unrecognized configuration parameter", so the Go layer strips them.
// The shared root .env keeps `schema=public` for the Node apps.
var prismaOnlyParams = []string{"schema"}

// cleanDSN removes Prisma-only query params from the shared DATABASE_URL.
func cleanDSN(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	q := u.Query()
	changed := false
	for _, p := range prismaOnlyParams {
		if q.Has(p) {
			q.Del(p)
			changed = true
		}
	}
	if !changed {
		return dsn, nil
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// Store is a pooled Postgres handle. Open with New.
type Store struct {
	pool *pgxpool.Pool
}

// New opens a pgx pool. dsn is the shared DATABASE_URL.
func New(ctx context.Context, dsn string) (*Store, error) {
	dsn, err := cleanDSN(dsn)
	if err != nil {
		return nil, err
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

// Close releases the pool.
func (s *Store) Close() {
	s.pool.Close()
}

// Ping reports whether the database is reachable.
func (s *Store) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.pool.Ping(ctx)
}

// MigrateUp applies the embedded goose baseline (tables + indexes, IF NOT EXISTS).
// Safe on Prisma-migrated databases (no-op) and on empty ones (bootstrap).
func MigrateUp(ctx context.Context, dsn string) error {
	dsn, err := cleanDSN(dsn)
	if err != nil {
		return err
	}
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.UpContext(ctx, sqlDB, migrations.Dir)
}

// constraints mirrors the Prisma foreign keys (names included). EnsureConstraints
// issues each ALTER, ignoring duplicate_object (42710) so it is idempotent on
// Prisma-managed databases.
var constraints = []string{
	`ALTER TABLE "Document" ADD CONSTRAINT "Document_ownerId_fkey" FOREIGN KEY ("ownerId") REFERENCES "User"("id") ON DELETE RESTRICT ON UPDATE CASCADE`,
	`ALTER TABLE "DocumentUpdate" ADD CONSTRAINT "DocumentUpdate_documentId_fkey" FOREIGN KEY ("documentId") REFERENCES "Document"("id") ON DELETE CASCADE ON UPDATE CASCADE`,
	`ALTER TABLE "DocumentSnapshot" ADD CONSTRAINT "DocumentSnapshot_documentId_fkey" FOREIGN KEY ("documentId") REFERENCES "Document"("id") ON DELETE CASCADE ON UPDATE CASCADE`,
	`ALTER TABLE "DocumentCollaborator" ADD CONSTRAINT "DocumentCollaborator_documentId_fkey" FOREIGN KEY ("documentId") REFERENCES "Document"("id") ON DELETE CASCADE ON UPDATE CASCADE`,
	`ALTER TABLE "DocumentCollaborator" ADD CONSTRAINT "DocumentCollaborator_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE`,
}

// EnsureConstraints creates the Prisma-equivalent foreign keys when missing.
func (s *Store) EnsureConstraints(ctx context.Context) error {
	for _, ddl := range constraints {
		if _, err := s.pool.Exec(ctx, ddl); err != nil {
			if isDuplicateObject(err) {
				continue
			}
			return err
		}
	}
	return nil
}
