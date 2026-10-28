package db_test

import (
	"context"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"long/internal/sqlc"
	"long/internal/testutil"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestDeletionMigrationAndFilters(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool := testutil.EmptyDatabase(t)
	sqlDB := stdlib.OpenDBFromPool(pool)
	t.Cleanup(func() { _ = sqlDB.Close() })
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, os.DirFS("migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 20261003101000); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
 INSERT INTO public.user(username,email) VALUES('alpha','alpha@example.com'),('beta','beta@example.com');
 INSERT INTO image(user_id,image_key,image_url,deleted_at) VALUES(1,'image','url',NOW()),(2,'other','url',NULL);
 INSERT INTO deletion(image_id,user_id,reason,status,created_at) VALUES(1,1,'literal_%','approved','2026-10-23'),(2,1,'second reason','active','2026-10-23'),(2,2,'third reason','rejected','2026-10-23');
 UPDATE image SET active_deletion_id=1 WHERE id=1;
 `); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	q := sqlc.New(pool)
	pending, err := q.GetUnindexedImages(ctx)
	if err != nil || len(pending) != 1 || pending[0].ImageID != 1 || pending[0].VersionID != 0 {
		t.Fatal("historical deletion was not eligible for index reconciliation", pending, err)
	}
	for _, tc := range []struct {
		status, search, pattern, sort, direction string
		id                                       int64
		offset                                   int32
		ids                                      []int64
		total                                    int64
	}{
		{"ACTIVE", "", "%", "CREATED_AT", "DESC", 0, 0, []int64{2}, 1},
		{"ALL", "", "%", "CREATED_AT", "DESC", 0, 0, []int64{3, 2}, 3},
		{"ALL", "", "%", "CREATED_AT", "DESC", 0, 2, []int64{1}, 3},
		{"ALL", "alpha", "%alpha%", "ID", "ASC", 0, 0, []int64{1, 2}, 2},
		{"ALL", "2", "%2%", "ID", "ASC", 2, 0, []int64{2, 3}, 2},
		{"APPROVED", "_%", `%\_\%%`, "PROCESSED_AT", "ASC", 0, 0, []int64{1}, 1},
		{"REJECTED", "", "%", "ID", "DESC", 0, 0, []int64{3}, 1},
	} {
		rows, err := q.AdminListDeletionRequests(ctx, sqlc.AdminListDeletionRequestsParams{Status: tc.status, Search: tc.search, Pattern: tc.pattern, SearchID: tc.id, SortBy: tc.sort, Direction: tc.direction, PageOffset: tc.offset, PageLimit: 2})
		if err != nil {
			t.Fatal(err)
		}
		ids := make([]int64, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ID)
		}
		total, err := q.AdminCountDeletionRequests(ctx, sqlc.AdminCountDeletionRequestsParams{Status: tc.status, Search: tc.search, Pattern: tc.pattern, SearchID: tc.id})
		if err != nil || total != tc.total || !reflect.DeepEqual(ids, tc.ids) {
			t.Fatalf("%+v: %v total%d %v", tc, ids, total, err)
		}
	}
	if _, err := provider.DownTo(ctx, 20261003101000); err != nil {
		t.Fatal(err)
	}
	var owner int64
	if err := pool.QueryRow(ctx, "SELECT user_id FROM deletion WHERE id=1").Scan(&owner); err != nil || owner != 1 {
		t.Fatal("rollback removed existing user_id", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal("reapply migration", err)
	}
}
