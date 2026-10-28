package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"long/internal/sqlc"
	"long/internal/testutil"
)

func TestRemoveDeletionOutboxMigrationPreservesDeletion(t *testing.T) {
	ctx := context.Background()
	pool := testutil.EmptyDatabase(t)
	sqlDB := stdlib.OpenDBFromPool(pool)
	t.Cleanup(func() { _ = sqlDB.Close() })
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, os.DirFS("migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 20261027010000); err != nil {
		t.Fatal(err)
	}
	testutil.Seed(t, pool)
	if _, err := pool.Exec(ctx, "INSERT INTO deletion(image_id,user_id,reason,status) VALUES(1,2,'duplicate','approved'); UPDATE image SET deleted_at=NOW(),active_deletion_id=1 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	before, err := sqlc.New(pool).GetDeletionRequestRecord(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 20261027020000); err != nil {
		t.Fatal(err)
	}
	var exists bool
	if err := pool.QueryRow(ctx, "SELECT to_regclass('public.deletion_outbox') IS NOT NULL").Scan(&exists); err != nil || exists {
		t.Fatal("outbox table was not removed", exists, err)
	}
	after, err := sqlc.New(pool).GetDeletionRequestRecord(ctx, 1)
	if err != nil || after != before {
		t.Fatal("table removal changed the deletion record", before, after, err)
	}
}
