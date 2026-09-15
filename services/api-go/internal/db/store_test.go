package db

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// testStore connects to DATABASE_URL, or skips when unreachable (unit runs
// without a DB skip). It self-bootstraps the schema (MigrateUp + constraints)
// so a fresh database — CI service included — works without Prisma.
func testStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	if err := MigrateUp(ctx, dsn); err != nil {
		t.Skipf("migrate failed (postgres unreachable?): %v", err)
	}
	store, err := New(ctx, dsn)
	if err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}
	if err := store.EnsureConstraints(ctx); err != nil {
		t.Fatalf("EnsureConstraints: %v", err)
	}
	t.Cleanup(store.Close)
	return store, ctx
}

func TestUserDocumentAccessFlow(t *testing.T) {
	store, ctx := testStore(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	ownerEmail := "go-owner-" + suffix + "@test.local"
	guestEmail := "go-guest-" + suffix + "@test.local"

	owner, err := store.CreateUser(ctx, ownerEmail, "argon2id$test")
	if err != nil {
		t.Fatalf("CreateUser owner: %v", err)
	}
	guest, err := store.CreateUser(ctx, guestEmail, "argon2id$test")
	if err != nil {
		t.Fatalf("CreateUser guest: %v", err)
	}

	var docID string
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if docID != "" {
			_, _ = store.DeleteDocument(ctx, docID, owner.ID)
		}
		_, _ = store.pool.Exec(ctx, `DELETE FROM "User" WHERE "id" = $1`, owner.ID)
		_, _ = store.pool.Exec(ctx, `DELETE FROM "User" WHERE "id" = $1`, guest.ID)
	})

	if got, err := store.FindUserByEmail(ctx, ownerEmail); err != nil || got == nil || got.ID != owner.ID {
		t.Fatalf("FindUserByEmail = %+v, %v", got, err)
	}

	doc, err := store.CreateDocument(ctx, owner.ID, "go test doc")
	if err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	docID = doc.ID

	// Guest has no access before sharing.
	if ok, err := store.CheckDocumentAccess(ctx, doc.ID, guest.ID); err != nil || ok {
		t.Fatalf("CheckDocumentAccess guest before share = %v, %v", ok, err)
	}
	if got, err := store.GetAccessibleDocument(ctx, doc.ID, guest.ID); err != nil || got != nil {
		t.Fatalf("GetAccessibleDocument guest before share = %+v, %v", got, err)
	}

	// Share → guest can open, owner flag false.
	if _, err := store.ShareDocument(ctx, doc.ID, owner.ID, guest.ID); err != nil {
		t.Fatalf("ShareDocument: %v", err)
	}
	if ok, err := store.CheckDocumentAccess(ctx, doc.ID, guest.ID); err != nil || !ok {
		t.Fatalf("CheckDocumentAccess guest after share = %v, %v", ok, err)
	}
	got, err := store.GetAccessibleDocument(ctx, doc.ID, guest.ID)
	if err != nil || got == nil || got.IsOwner {
		t.Fatalf("GetAccessibleDocument guest = %+v, %v", got, err)
	}
	mine, err := store.GetAccessibleDocument(ctx, doc.ID, owner.ID)
	if err != nil || mine == nil || !mine.IsOwner {
		t.Fatalf("GetAccessibleDocument owner = %+v, %v", mine, err)
	}

	// Guest rename must not touch the row (owner-scoped write).
	if ok, err := store.RenameDocument(ctx, doc.ID, guest.ID, "hijacked"); err != nil || ok {
		t.Fatalf("RenameDocument guest = %v, %v", ok, err)
	}
	if ok, err := store.RenameDocument(ctx, doc.ID, owner.ID, "renamed"); err != nil || !ok {
		t.Fatalf("RenameDocument owner = %v, %v", ok, err)
	}

	docs, err := store.ListDocuments(ctx, guest.ID)
	if err != nil || len(docs) != 1 || docs[0].IsOwner {
		t.Fatalf("ListDocuments guest = %+v, %v", docs, err)
	}

	if err := store.UnshareDocument(ctx, doc.ID, guest.ID); err != nil {
		t.Fatalf("UnshareDocument: %v", err)
	}
	if ok, err := store.CheckDocumentAccess(ctx, doc.ID, guest.ID); err != nil || ok {
		t.Fatalf("CheckDocumentAccess guest after unshare = %v, %v", ok, err)
	}
}

func TestOpLogCompactRoundTrip(t *testing.T) {
	store, ctx := testStore(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	owner, err := store.CreateUser(ctx, "go-oplog-"+suffix+"@test.local", "argon2id$test")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	doc, err := store.CreateDocument(ctx, owner.ID, "go oplog doc")
	if err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = store.DeleteDocument(ctx, doc.ID, owner.ID)
		_, _ = store.pool.Exec(ctx, `DELETE FROM "User" WHERE "id" = $1`, owner.ID)
	})

	// Empty log: clock starts at -1, snapshot miss.
	if clock, err := store.MaxClock(ctx, doc.ID); err != nil || clock != -1 {
		t.Fatalf("MaxClock empty = %d, %v", clock, err)
	}
	if snap, err := store.LatestSnapshot(ctx, doc.ID); err != nil || snap != nil {
		t.Fatalf("LatestSnapshot empty = %+v, %v", snap, err)
	}
	if pruned, err := store.Compact(ctx, doc.ID, []byte{1, 2, 3}); err != nil || pruned != 0 {
		t.Fatalf("Compact empty = %d, %v", pruned, err)
	}

	// Append 3 updates, clocks 0..2.
	for clock := 0; clock < 3; clock++ {
		if err := store.AppendUpdate(ctx, doc.ID, []byte{byte(clock)}, clock); err != nil {
			t.Fatalf("AppendUpdate %d: %v", clock, err)
		}
	}
	if clock, err := store.MaxClock(ctx, doc.ID); err != nil || clock != 2 {
		t.Fatalf("MaxClock = %d, %v", clock, err)
	}
	updates, err := store.ListUpdatesAfter(ctx, doc.ID, 0)
	if err != nil || len(updates) != 3 {
		t.Fatalf("ListUpdatesAfter = %d, %v", len(updates), err)
	}

	// Compact: snapshot commit → prune in one tx (invariant #3).
	if pruned, err := store.Compact(ctx, doc.ID, []byte{9, 9}); err != nil || pruned != 3 {
		t.Fatalf("Compact = %d, %v", pruned, err)
	}
	snap, err := store.LatestSnapshot(ctx, doc.ID)
	if err != nil || snap == nil || len(snap.State) != 2 {
		t.Fatalf("LatestSnapshot after compact = %+v, %v", snap, err)
	}
	rest, err := store.ListUpdatesAfter(ctx, doc.ID, snap.UpToUpdateID)
	if err != nil || len(rest) != 0 {
		t.Fatalf("ListUpdatesAfter snapshot = %d, %v", len(rest), err)
	}

	// Audit projections see 1 snapshot, 0 live updates.
	snaps, err := store.AuditSnapshots(ctx, doc.ID)
	if err != nil || len(snaps) != 1 || snaps[0].Bytes != 2 {
		t.Fatalf("AuditSnapshots = %+v, %v", snaps, err)
	}
	live, err := store.AuditUpdates(ctx, doc.ID)
	if err != nil || len(live) != 0 {
		t.Fatalf("AuditUpdates = %+v, %v", live, err)
	}
}
