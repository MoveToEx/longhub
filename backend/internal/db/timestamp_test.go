package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"testing"
	"time"

	"long/internal/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// TEST_DATABASE_URL must allow creating databases; each case uses its own temporary database.
func TestTimestampMigration(t *testing.T) {
	for _, timezone := range []string{"Asia/Shanghai", "UTC"} {
		t.Run(timezone, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			pool, sqlDB := timestampTestDatabase(t, ctx, timezone)
			migrations, err := fs.Sub(embedMigrations, "migrations")
			if err != nil {
				t.Fatal(err)
			}
			provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := provider.UpTo(ctx, 20261003100938); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `
INSERT INTO public.user (email, username) VALUES ('test@example.com', 'test');
INSERT INTO image (user_id, image_key, image_url) VALUES (1, 'test', 'https://example.com/test');
INSERT INTO version (image_id, text, rating, user_id) VALUES (1, 'test', 'none', 1);
UPDATE image SET current_version_id = 1, indexed_version = 1;
INSERT INTO user_favorite (user_id, image_id) VALUES (1, 1);
INSERT INTO appkey (user_id, permission, key) VALUES (1, 0, 'test');
INSERT INTO webhook (user_id, label, endpoint, secret) VALUES (1, 'test', 'https://example.com', 'test');
INSERT INTO webauthn_passkey (id, user_id, public_key, sign_count, flags, aaguid)
    VALUES ('test', 1, 'test', 0, 0, '00000000-0000-0000-0000-000000000000');
INSERT INTO deletion (image_id, user_id, reason, status) VALUES (1, 1, 'test', 'active');
INSERT INTO upload_session (user_id, key) VALUES (1, 'test');
`); err != nil {
				t.Fatal(err)
			}

			rows, err := pool.Query(ctx, `SELECT table_name, column_name FROM information_schema.columns
WHERE table_schema = 'public' AND data_type = 'timestamp without time zone'
    AND table_name NOT IN ('user_identifier', 'goose_db_version') ORDER BY table_name, column_name`)
			if err != nil {
				t.Fatal(err)
			}
			type column struct {
				table, name string
				null        bool
			}
			var columns []column
			for rows.Next() {
				var c column
				if err := rows.Scan(&c.table, &c.name); err != nil {
					t.Fatal(err)
				}
				c.null = c.name == "deleted_at" || c.table == "appkey" && c.name == "last_activated_at"
				columns = append(columns, c)
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			if len(columns) != 15 {
				t.Fatalf("expected 15 legacy timestamp columns, got %d", len(columns))
			}
			const wallClock = "2026-10-03 12:34:56.123456"
			for _, c := range columns {
				if c.null {
					continue
				}
				query := "UPDATE " + pgx.Identifier{c.table}.Sanitize() + " SET " + pgx.Identifier{c.name}.Sanitize() + " = $1::timestamp"
				if _, err := pool.Exec(ctx, query, wallClock); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := provider.UpTo(ctx, 20261003101000); err != nil {
				t.Fatal(err)
			}

			location, err := time.LoadLocation(timezone)
			if err != nil {
				t.Fatal(err)
			}
			want := time.Date(2026, 10, 3, 12, 34, 56, 123456000, location)
			for _, c := range columns {
				query := "SELECT " + pgx.Identifier{c.name}.Sanitize() + " FROM " + pgx.Identifier{c.table}.Sanitize()
				var value pgtype.Timestamptz
				if err := pool.QueryRow(ctx, query).Scan(&value); err != nil {
					t.Fatalf("%s.%s: %v", c.table, c.name, err)
				}
				if value.Valid != !c.null || value.Valid && !value.Time.Equal(want) {
					t.Fatalf("%s.%s: unexpected timestamp %+v, want %v (null=%v)", c.table, c.name, value, want, c.null)
				}
			}
			var legacyCount int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns
WHERE table_schema = 'public' AND data_type = 'timestamp without time zone'
    AND table_name <> 'goose_db_version'`).Scan(&legacyCount); err != nil {
				t.Fatal(err)
			}
			if legacyCount != 0 {
				t.Fatalf("%d columns still lack timezones", legacyCount)
			}

			// The same instant must survive a different database session timezone and JSON serialization.
			if _, err := pool.Exec(ctx, "SET TIME ZONE 'UTC'"); err != nil {
				t.Fatal(err)
			}
			image, err := sqlc.New(pool).GetImage(ctx, 1)
			if err != nil {
				t.Fatal(err)
			}
			body, err := json.Marshal(image)
			if err != nil {
				t.Fatal(err)
			}
			var response struct {
				CreatedAt      time.Time `json:"createdAt"`
				UserIdentifier struct {
					CreatedAt time.Time `json:"createdAt"`
				} `json:"userIdentifier"`
			}
			if err := json.Unmarshal(body, &response); err != nil {
				t.Fatalf("timestamp JSON must include a timezone: %s: %v", body, err)
			}
			if !response.CreatedAt.Equal(want) || !response.UserIdentifier.CreatedAt.Equal(want) {
				t.Fatalf("JSON timestamps represent the wrong instant: %s, want %v", body, want)
			}
			if image.CreatedAt.Time.Unix() != want.Unix() {
				t.Fatal("search indexing would use the wrong Unix timestamp")
			}
			var indexed pgtype.Int8
			if err := pool.QueryRow(ctx, "SELECT indexed_version FROM image").Scan(&indexed); err != nil {
				t.Fatal(err)
			}
			if indexed.Valid {
				t.Fatal("existing search documents must be scheduled for reindexing")
			}
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var defaultIsNow bool
			err = tx.QueryRow(ctx, `INSERT INTO public.user (email, username) VALUES ('new@example.com', 'new')
RETURNING created_at = NOW()`).Scan(&defaultIsNow)
			_ = tx.Rollback(ctx)
			if err != nil || !defaultIsNow {
				t.Fatalf("new timestamps must preserve NOW(): %v", err)
			}

			if _, err := pool.Exec(ctx, "SELECT set_config('TimeZone', $1, false)", timezone); err != nil {
				t.Fatal(err)
			}
			if _, err := provider.Down(ctx); err != nil {
				t.Fatal(err)
			}
			pool.Reset()
			for _, c := range columns {
				query := "SELECT " + pgx.Identifier{c.name}.Sanitize() + " FROM " + pgx.Identifier{c.table}.Sanitize()
				var value pgtype.Timestamp
				if err := pool.QueryRow(ctx, query).Scan(&value); err != nil {
					t.Fatal(err)
				}
				if value.Valid != !c.null || value.Valid && value.Time.Format("2006-01-02 15:04:05.999999") != wallClock {
					t.Fatalf("rollback changed %s.%s: %+v", c.table, c.name, value)
				}
			}
		})
	}
}

func timestampTestDatabase(t *testing.T, ctx context.Context, timezone string) (*pgxpool.Pool, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL migration tests")
	}
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Close(context.Background()) })
	database := fmt.Sprintf("longhub_timestamp_test_%d", time.Now().UnixNano())
	quotedDatabase := pgx.Identifier{database}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+quotedDatabase); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), "DROP DATABASE "+quotedDatabase); err != nil {
			t.Errorf("drop test database: %v", err)
		}
	})
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.Database = database
	config.ConnConfig.RuntimeParams["TimeZone"] = timezone
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB := stdlib.OpenDBFromPool(pool)
	t.Cleanup(func() {
		_ = sqlDB.Close()
		pool.Close()
	})
	return pool, sqlDB
}
