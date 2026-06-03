package queue

import (
	"context"
	"long/internal/config"
	"long/internal/db"
	"long/internal/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/meilisearch/meilisearch-go"
	"github.com/riverqueue/river"
)

type IndexArgs struct{}

func (IndexArgs) Kind() string {
	return "ms-index"
}

type IndexWorker struct {
	river.WorkerDefaults[IndexArgs]
}

type IndexPayloadItem struct {
	ID     int64    `json:"id"`
	Text   string   `json:"text"`
	Tags   []string `json:"tags"`
	Rating int      `json:"rating"`
}

func toInt(r sqlc.Rating) int {
	switch r {
	case "violent":
		return 3
	case "moderate":
		return 2
	case "none":
		return 1
	}
	return -1
}

func (w *IndexWorker) Work(ctx context.Context, job *river.Job[IndexArgs]) error {
	ms := config.MeiliSearch()

	err := db.Transaction(ctx, func(tx *sqlc.Queries) error {
		versions, err := tx.GetUnindexedVersions(ctx)

		if err != nil {
			return err
		}

		doc := []IndexPayloadItem{}

		for i := range versions {
			doc = append(doc, IndexPayloadItem{
				ID:     versions[i].ImageID,
				Text:   versions[i].Text,
				Tags:   versions[i].Tags,
				Rating: toInt(versions[i].Rating),
			})
		}

		_, err = ms.Index("images").AddDocuments(doc, &meilisearch.DocumentOptions{
			PrimaryKey: new("id"),
		})

		if err != nil {
			return err
		}

		for i := range versions {
			err := tx.CommitUnindexedVersions(ctx, sqlc.CommitUnindexedVersionsParams{
				ID: versions[i].ImageID,
				IndexedVersion: pgtype.Int8{
					Valid: true,
					Int64: versions[i].ID,
				},
			})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}
