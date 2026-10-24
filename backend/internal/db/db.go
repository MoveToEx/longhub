package db

import (
	"context"
	"database/sql"
	"embed"

	"long/internal/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
)

var conn *pgxpool.Pool

//go:embed migrations/*.sql
var embedMigrations embed.FS

func Migrate(ctx context.Context, conn *sql.DB) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("pgx"); err != nil {
		return err
	}

	return goose.UpContext(ctx, conn, "migrations")
}

func Init(ctx context.Context, pool *pgxpool.Pool) {
	conn = pool
}

func Query() *sqlc.Queries {
	return sqlc.New(conn)
}

func Transaction(ctx context.Context, f func(qtx *sqlc.Queries) error) error {
	return transaction(ctx, pgx.TxOptions{}, f)
}

// ReadSnapshot keeps all reads, including paged exports, on one database snapshot.
func ReadSnapshot(ctx context.Context, f func(qtx *sqlc.Queries) error) error {
	return transaction(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}, f)
}

func transaction(ctx context.Context, options pgx.TxOptions, f func(*sqlc.Queries) error) error {
	tx, err := conn.BeginTx(ctx, options)

	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := Query().WithTx(tx)

	if err := f(qtx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
