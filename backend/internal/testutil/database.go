// Package testutil provides isolated databases for integration tests.
package testutil

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"long/internal/db"
	"long/internal/deletion"
	"long/internal/sqlc"
	"os"
	"testing"
	"time"
)

func Database(t *testing.T) (*pgxpool.Pool, deletion.Transaction) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL for isolated PostgreSQL integration tests")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("longhub_deletion_test_%d", time.Now().UnixNano())
	quoted := pgx.Identifier{name}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+quoted); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Database = name
	cfg.MaxConns = 8
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		if _, err := admin.Exec(ctx, "DROP DATABASE "+quoted); err != nil {
			t.Error(err)
		}
		_ = admin.Close(ctx)
	})
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	transact := func(ctx context.Context, run func(*sqlc.Queries) error) error {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		if err := run(sqlc.New(tx)); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	return pool, transact
}

func Seed(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
 INSERT INTO public.user(username,email,permission) VALUES ('admin','admin@example.com',8),('reporter','reporter@example.com',4),('other','other@example.com',4);
 INSERT INTO image(user_id,image_key,image_url) VALUES(2,'test','https://example.com/test.png'),(2,'empty','https://example.com/empty.png');
 INSERT INTO version(image_id,user_id,text,rating) VALUES(1,2,'original text','none');
 UPDATE image SET current_version_id=1 WHERE id=1;
 INSERT INTO user_favorite(user_id,image_id,shortcut) VALUES(2,1,'favorite');
 INSERT INTO tag(name) VALUES('example');
 INSERT INTO version_tag(version_id,tag_id) VALUES(1,1);
 `)
	if err != nil {
		t.Fatal(err)
	}
}
