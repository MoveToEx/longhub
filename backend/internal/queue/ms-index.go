package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"long/internal/config"
	"long/internal/db"
	"long/internal/sqlc"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/meilisearch/meilisearch-go"
)

type IndexArgs struct {
	ImageID   int64 `json:"imageId"`
	VersionID int64 `json:"versionId"`
}

func NewIndexTask(imageID, versionID int64) (*asynq.Task, error) {
	payload, err := json.Marshal(IndexArgs{
		ImageID:   imageID,
		VersionID: versionID,
	})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeIndex, payload), nil
}

func EnqueueIndex(ctx context.Context, imageID, versionID int64) error {
	task, err := NewIndexTask(imageID, versionID)
	if err != nil {
		return err
	}

	_, err = client.EnqueueContext(ctx, task, asynq.Unique(time.Minute))
	if errors.Is(err, asynq.ErrDuplicateTask) {
		return nil
	}
	return err
}

func NewIndexReconcileTask() *asynq.Task {
	return asynq.NewTask(TypeIndexReconcile, nil)
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

func HandleIndexTask(ctx context.Context, task *asynq.Task) error {
	var args IndexArgs
	if err := json.Unmarshal(task.Payload(), &args); err != nil {
		return err
	}

	return db.Transaction(ctx, func(tx *sqlc.Queries) error {
		version, err := tx.GetUnindexedVersion(ctx, sqlc.GetUnindexedVersionParams{
			ImageID:   args.ImageID,
			VersionID: args.VersionID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}

		document := IndexPayloadItem{
			ID:        version.ImageID,
			Text:      version.Text,
			Tags:      version.Tags,
			Rating:    toInt(version.Rating),
			UserID:    version.Image.UserID,
			Uploader:  version.Uploader,
			CreatedAt: version.Image.CreatedAt.Time.Unix(),
		}

		meili := config.MeiliSearch()
		meiliTask, err := meili.Index("images").AddDocumentsWithContext(
			ctx,
			[]IndexPayloadItem{document},
			&meilisearch.DocumentOptions{PrimaryKey: new("id")},
		)
		if err != nil {
			return err
		}

		result, err := meili.WaitForTaskWithContext(ctx, meiliTask.TaskUID, 50*time.Millisecond)
		if err != nil {
			return err
		}
		if result.Status != meilisearch.TaskStatusSucceeded {
			return fmt.Errorf("indexing image %d finished with status %q: %s", args.ImageID, result.Status, result.Error.Message)
		}

		return tx.CommitUnindexedVersion(ctx, sqlc.CommitUnindexedVersionParams{
			ImageID: args.ImageID,
			VersionID: pgtype.Int8{
				Int64: args.VersionID,
				Valid: true,
			},
		})
	})
}

func HandleIndexReconcileTask(ctx context.Context, _ *asynq.Task) error {
	images, err := db.Query().GetUnindexedImages(ctx)
	if err != nil {
		return err
	}

	for _, image := range images {
		if err := EnqueueIndex(ctx, image.ImageID, image.VersionID); err != nil {
			return err
		}
	}

	return nil
}
