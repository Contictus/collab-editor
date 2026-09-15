package db

import (
	"context"
)

// OpLog mirrors ws-server/src/persistence.ts: append-only updates, latest
// snapshot + replay of newer updates (load-on-open), and snapshot compaction.
//
// INVARIANT #3: Compact commits the snapshot BEFORE pruning covered updates —
// both in one transaction, so no crash window prunes without a durable snapshot.

// LatestSnapshot returns the newest snapshot for a document, or nil when none.
func (s *Store) LatestSnapshot(ctx context.Context, documentID string) (*Snapshot, error) {
	var snap Snapshot
	err := s.pool.QueryRow(ctx,
		`SELECT "id", "documentId", "state", "upToUpdateId", "createdAt"
		 FROM "DocumentSnapshot" WHERE "documentId" = $1 ORDER BY "id" DESC LIMIT 1`, documentID,
	).Scan(&snap.ID, &snap.DocumentID, &snap.State, &snap.UpToUpdateID, &snap.CreatedAt)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return &snap, nil
}

// ListUpdatesAfter returns updates recorded after upToUpdateId (exclusive),
// oldest first. Pass 0 to read the whole log (no snapshot yet).
func (s *Store) ListUpdatesAfter(ctx context.Context, documentID string, upToUpdateID int64) ([][]byte, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT "update" FROM "DocumentUpdate"
		 WHERE "documentId" = $1 AND "id" > $2 ORDER BY "id" ASC`, documentID, upToUpdateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out [][]byte
	for rows.Next() {
		var u []byte
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// MaxClock returns the highest clock assigned for a document, or -1 when the
// log is empty — so the next clock is MaxClock()+1 (mirrors loadInto).
func (s *Store) MaxClock(ctx context.Context, documentID string) (int, error) {
	var max *int
	err := s.pool.QueryRow(ctx,
		`SELECT MAX("clock") FROM "DocumentUpdate" WHERE "documentId" = $1`, documentID,
	).Scan(&max)
	if err != nil {
		return 0, err
	}
	if max == nil {
		return -1, nil
	}
	return *max, nil
}

// LoadResult mirrors LoadResult in persistence.ts.
type LoadResult struct {
	Clock         int
	SinceSnapshot int
}

// AppendUpdate appends one binary Yjs update to the op log (never pruned here).
func (s *Store) AppendUpdate(ctx context.Context, documentID string, update []byte, clock int) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO "DocumentUpdate" ("documentId", "update", "clock") VALUES ($1, $2, $3)`,
		documentID, update, clock)
	return err
}

// Compact checkpoints the authoritative state, then prunes covered updates —
// snapshot insert FIRST, prune second, in one transaction (INVARIANT #3).
// Returns the pruned row count (0 when the log is empty). state must be the
// full-state encoding (Y.encodeStateAsUpdate equivalent); callers in F5 supply it.
func (s *Store) Compact(ctx context.Context, documentID string, state []byte) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var lastID *int64
	err = tx.QueryRow(ctx,
		`SELECT MAX("id") FROM "DocumentUpdate" WHERE "documentId" = $1`, documentID,
	).Scan(&lastID)
	if err != nil {
		return 0, err
	}
	if lastID == nil {
		return 0, tx.Commit(ctx)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO "DocumentSnapshot" ("documentId", "state", "upToUpdateId")
		 VALUES ($1, $2, $3)`, documentID, state, *lastID); err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx,
		`DELETE FROM "DocumentUpdate" WHERE "documentId" = $1 AND "id" <= $2`, documentID, *lastID)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// AuditSnapshots lists snapshots oldest→newest with byte sizes (no blobs).
func (s *Store) AuditSnapshots(ctx context.Context, documentID string) ([]SnapshotMeta, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT "id", "upToUpdateId", OCTET_LENGTH("state"), "createdAt"
		 FROM "DocumentSnapshot" WHERE "documentId" = $1 ORDER BY "id" ASC`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SnapshotMeta
	for rows.Next() {
		var m SnapshotMeta
		if err := rows.Scan(&m.ID, &m.UpToUpdateID, &m.Bytes, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// AuditUpdates lists surviving op-log rows oldest→newest with byte sizes.
func (s *Store) AuditUpdates(ctx context.Context, documentID string) ([]UpdateMeta, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT "id", "clock", OCTET_LENGTH("update"), "createdAt"
		 FROM "DocumentUpdate" WHERE "documentId" = $1 ORDER BY "id" ASC`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UpdateMeta
	for rows.Next() {
		var m UpdateMeta
		if err := rows.Scan(&m.ID, &m.Clock, &m.Bytes, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ReplaySnapshots returns full snapshot blobs oldest→newest (replay endpoint).
func (s *Store) ReplaySnapshots(ctx context.Context, documentID string) ([]Snapshot, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT "id", "documentId", "state", "upToUpdateId", "createdAt"
		 FROM "DocumentSnapshot" WHERE "documentId" = $1 ORDER BY "id" ASC`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Snapshot
	for rows.Next() {
		var snap Snapshot
		if err := rows.Scan(&snap.ID, &snap.DocumentID, &snap.State, &snap.UpToUpdateID, &snap.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, snap)
	}
	return out, rows.Err()
}

// ReplayUpdates returns full update blobs oldest→newest (replay endpoint).
func (s *Store) ReplayUpdates(ctx context.Context, documentID string) ([]OpUpdate, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT "id", "documentId", "update", "clock", "createdAt"
		 FROM "DocumentUpdate" WHERE "documentId" = $1 ORDER BY "id" ASC`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OpUpdate
	for rows.Next() {
		var u OpUpdate
		if err := rows.Scan(&u.ID, &u.DocumentID, &u.Update, &u.Clock, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
