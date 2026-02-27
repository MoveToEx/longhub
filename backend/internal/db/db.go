package db

import (
	"context"
	"database/sql"
	"embed"

	_ "long/internal/db/migrations"
	"long/internal/sqlc"

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
	tx, err := conn.Begin(ctx)

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
