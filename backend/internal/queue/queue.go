package queue

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

var client *river.Client[pgx.Tx]
var ctx context.Context

func Init(conn *pgxpool.Pool) error {
	workers := river.NewWorkers()

	if err := river.AddWorkerSafely(workers, &HashWorker{}); err != nil {
		return err
	}

	var err error

	client, err = river.NewClient(riverpgxv5.New(conn), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {
				MaxWorkers: 100,
			},
		},
		Workers: workers,
	})

	if err != nil {
		return err
	}

	ctx = context.Background()

	if err := client.Start(ctx); err != nil {
		return err
	}

	return nil
}

func Stop() error {
	return client.Stop(ctx)
}
