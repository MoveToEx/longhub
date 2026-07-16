package queue

import (
	"context"
	"long/internal/config"
	"long/internal/db"
	"long/internal/sqlc"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/meilisearch/meilisearch-go"
)

func NewIndexTask() *asynq.Task {
	return asynq.NewTask(TypeIndex, nil)
}

type IndexPayloadItem struct {
	ID        int64    `json:"id"`
	Text      string   `json:"text"`
	Tags      []string `json:"tags"`
	Rating    int      `json:"rating"`
	UserID    int64    `json:"userId"`
	Uploader  string   `json:"uploader"`
	CreatedAt int64    `json:"createdAt"`
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

func HandleIndexTask(ctx context.Context, _ *asynq.Task) error {
	ms := config.MeiliSearch()

	err := db.Transaction(ctx, func(tx *sqlc.Queries) error {
		versions, err := tx.GetUnindexedVersions(ctx)

		if err != nil {
			return err
		}

		if len(versions) == 0 {
			return nil
		}

		doc := []IndexPayloadItem{}

		for i := range versions {
			doc = append(doc, IndexPayloadItem{
				ID:        versions[i].ImageID,
				Text:      versions[i].Text,
				Tags:      versions[i].Tags,
				Rating:    toInt(versions[i].Rating),
				UserID:    versions[i].Image.UserID,
				Uploader:  versions[i].Uploader,
				CreatedAt: versions[i].Image.CreatedAt.Time.Unix(),
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
