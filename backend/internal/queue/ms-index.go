package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"long/internal/config"
	"long/internal/db"
	"long/internal/deletion"
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

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return syncImageIndex(ctx, args, db.Transaction, writeImageIndex)
}

// Hold the same image lock as moderation until Meili acknowledges the write.
// Jobs always use current database state, including jobs queued before deletion.
func syncImageIndex(ctx context.Context, args IndexArgs, transact deletion.Transaction, write func(context.Context, int64, *IndexPayloadItem) error) error {
	return transact(ctx, func(tx *sqlc.Queries) error {
		image, err := tx.LockImageForDeletion(ctx, args.ImageID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}

		if image.DeletedAt.Valid || !image.CurrentVersionID.Valid {
			if err := write(ctx, image.ID, nil); err != nil {
				return err
			}
			return tx.CommitImageIndexRemoval(ctx, image.ID)
		}
		if args.VersionID != 0 && image.IndexedVersion.Valid && image.IndexedVersion.Int64 == image.CurrentVersionID.Int64 {
			return nil
		}
		snapshot, err := tx.GetImageSnapshot(ctx, image.ID)
		if err != nil {
			return err
		}
		document := &IndexPayloadItem{
			ID: image.ID, Text: snapshot.Text, Tags: snapshot.Tags,
			Rating: toInt(sqlc.Rating(snapshot.Rating)), UserID: snapshot.UserID,
			Uploader: snapshot.Username, CreatedAt: snapshot.CreatedAt.Time.Unix(),
		}
		if err := write(ctx, image.ID, document); err != nil {
			return err
		}
		return tx.CommitUnindexedVersion(ctx, sqlc.CommitUnindexedVersionParams{
			ImageID: image.ID,
			VersionID: pgtype.Int8{
				Int64: image.CurrentVersionID.Int64,
				Valid: true,
			},
		})
	})
}

func writeImageIndex(ctx context.Context, imageID int64, document *IndexPayloadItem) error {
	meili := config.MeiliSearch()
	var task *meilisearch.TaskInfo
	var err error
	if document == nil {
		task, err = meili.Index("images").DeleteDocumentWithContext(ctx, fmt.Sprint(imageID), nil)
	} else {
		task, err = meili.Index("images").AddDocumentsWithContext(ctx, []IndexPayloadItem{*document}, &meilisearch.DocumentOptions{PrimaryKey: new("id")})
	}
	if err != nil {
		return err
	}
	result, err := meili.WaitForTaskWithContext(ctx, task.TaskUID, 50*time.Millisecond)
	if err != nil {
		return err
	}
	if result.Status != meilisearch.TaskStatusSucceeded {
		return fmt.Errorf("synchronizing image %d finished with status %q: %s", imageID, result.Status, result.Error.Message)
	}
	return nil
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
