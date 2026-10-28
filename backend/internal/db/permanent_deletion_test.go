package db_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"long/internal/sqlc"
	"long/internal/testutil"
)

func TestPermanentDeletionMigration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.EmptyDatabase(t)
	sqlDB := stdlib.OpenDBFromPool(pool)
	t.Cleanup(func() { _ = sqlDB.Close() })
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, os.DirFS("migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 20261027000000); err != nil {
		t.Fatal(err)
	}
	testutil.Seed(t, pool)
	if _, err := pool.Exec(ctx, `
INSERT INTO deletion(image_id,user_id,reason,status,processed_at,processed_by)
VALUES(1,2,'approved','approved',NOW(),1),(2,2,'old decision','revoked',NOW(),1),
      (2,2,'pending','active',NULL,NULL);
UPDATE image SET deleted_at=NOW(),active_deletion_id=1 WHERE id=1;
`); err != nil {
		t.Fatal(err)
	}
	q := sqlc.New(pool)
	before, err := q.GetSyncVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 20261027010000); err != nil {
		t.Fatal(err)
	}
	row, err := q.GetDeletionRequestRecord(ctx, 2)
	if err != nil || row.Status != sqlc.DeletionStatusRejected || row.ImageDeletedAt.Valid || !row.ProcessedAt.Valid || row.ProcessedBy.Int64 != 1 {
		t.Fatal("historical decision or active image changed", row, err)
	}
	var statuses int
	if err := pool.QueryRow(ctx, "SELECT cardinality(enum_range(NULL::deletion_status))").Scan(&statuses); err != nil || statuses != 3 {
		t.Fatal("unexpected deletion statuses", statuses, err)
	}
	after, err := q.GetSyncVersion(ctx)
	if err != nil || !before.Time.Equal(after.Time) {
		t.Fatal("migration changed the catalog version", before, after, err)
	}
	assertPermanent := func() {
		t.Helper()
		_, err := pool.Exec(ctx, "UPDATE image SET deleted_at=NULL,active_deletion_id=NULL WHERE id=1")
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
			t.Fatal("deleted image could be reinstated", err)
		}
		image, err := q.LockImageForDeletion(ctx, 1)
		if err != nil || !image.DeletedAt.Valid || image.ActiveDeletionID.Int64 != 1 {
			t.Fatal("deletion tombstone changed", image, err)
		}
	}
	assertPermanent()
	if _, err := pool.Exec(ctx, "UPDATE image SET indexed_version=0 WHERE id=1"); err != nil {
		t.Fatal("permanent deletion blocked indexing", err)
	}
	if _, err := provider.Down(ctx); err != nil {
		t.Fatal("rollback migration", err)
	}
	if _, err := provider.UpTo(ctx, 20261027010000); err != nil {
		t.Fatal("reapply migration", err)
	}
	assertPermanent()
}
