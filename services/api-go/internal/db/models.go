package db

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Models mirror the Prisma schema 1:1 (column for column). IDs are TEXT;
// Node generates cuid() values, Go generates random hex — the column accepts
// both, so mixed fleets interoperate.

// User mirrors model User.
type User struct {
	ID        string
	Email     string
	Password  string // argon2id hash (F2 verifies with the same PHC format)
	CreatedAt time.Time
}

// Document mirrors model Document.
type Document struct {
	ID        string
	Title     string
	OwnerID   string
	PublicID  *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DocumentSummary mirrors DocumentSummary in document-service.ts.
type DocumentSummary struct {
	ID        string
	Title     string
	UpdatedAt time.Time
	IsOwner   bool
}

// Collaborator mirrors CollaboratorSummary in document-service.ts.
// Role is 'editor' (read+write) or 'viewer' (read-only).
type Collaborator struct {
	UserID    string
	Email     string
	Role      string
	CreatedAt time.Time
}

// Snapshot mirrors model DocumentSnapshot.
type Snapshot struct {
	ID           int64
	DocumentID   string
	State        []byte
	UpToUpdateID int64
	CreatedAt    time.Time
}

// OpUpdate mirrors model DocumentUpdate.
type OpUpdate struct {
	ID         int64
	DocumentID string
	Update     []byte
	Clock      int
	CreatedAt  time.Time
}

// UpdateMeta and SnapshotMeta are the audit-list projections (byte counts, no blobs).
type UpdateMeta struct {
	ID        int64
	Clock     int
	Bytes     int
	CreatedAt time.Time
}

// SnapshotMeta is the audit-list projection for snapshots.
type SnapshotMeta struct {
	ID           int64
	UpToUpdateID int64
	Bytes        int
	CreatedAt    time.Time
}

// newID generates a TEXT primary key (24 hex chars, "c" prefix like cuid).
func newID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "c" + hex.EncodeToString(b[:]), nil
}
