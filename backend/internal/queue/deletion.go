package queue

import (
	"context"
	"errors"
	"time"

	"github.com/hibiken/asynq"
	"long/internal/db"
	"long/internal/webhook"
)

// EnqueueDeletion schedules webhook delivery and search removal after the deletion commits.
func EnqueueDeletion(ctx context.Context, event webhook.DeletionEvent) error {
	if client == nil {
		return errors.New("queue is not initialized")
	}
	task, err := newDispatchTask(DispatchArgs{
		ID: event.Data.ImageID, EventType: db.WebhookDeletionEvent, Deletion: &event,
	})
	if err != nil {
		return err
	}
	indexErr := EnqueueIndex(ctx, event.Data.ImageID, 0)
	_, dispatchErr := client.EnqueueContext(ctx, task, asynq.Unique(time.Hour))
	if errors.Is(dispatchErr, asynq.ErrDuplicateTask) {
		dispatchErr = nil
	}
	return errors.Join(indexErr, dispatchErr)
}
