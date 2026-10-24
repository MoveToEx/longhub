package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hibiken/asynq"
	"long/internal/db"
	"long/internal/deletion"
	"long/internal/sqlc"
	"time"
)

func NewDeletionOutboxTask() *asynq.Task { return asynq.NewTask(TypeDeletionOutbox, nil) }

func WakeDeletionOutbox(ctx context.Context) error {
	if client == nil {
		return nil
	}
	_, err := client.EnqueueContext(ctx, NewDeletionOutboxTask(), asynq.Unique(time.Second))
	if errors.Is(err, asynq.ErrDuplicateTask) {
		return nil
	}
	return err
}

type enqueueTask func(context.Context, *asynq.Task, ...asynq.Option) (*asynq.TaskInfo, error)

// A crash after enqueue is safe: stable task IDs and retained completions deduplicate retries.
func dispatchDeletionOutbox(ctx context.Context, transact deletion.Transaction, enqueue enqueueTask) error {
	return transact(ctx, func(q *sqlc.Queries) error {
		rows, err := q.LockPendingDeletionOutbox(ctx)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if row.Kind == "search" {
				task, err := NewIndexTask(row.ImageID, 0)
				if err != nil {
					return err
				}
				_, err = enqueue(ctx, task, asynq.TaskID(fmt.Sprintf("deletion-search-%d", row.ID)), asynq.Retention(24*time.Hour))
				if err != nil && !errors.Is(err, asynq.ErrTaskIDConflict) {
					return err
				}
			} else {
				var event deletion.Event
				if err := json.Unmarshal(row.Payload, &event); err != nil {
					return err
				}
				webhooks, err := q.GetWebhooksByEvent(ctx, sqlc.GetWebhooksByEventParams{AfterID: row.AfterWebhookID, EventType: db.WebhookDeletionEvent, FailureCount: WebhookMaxFailures, PageLimit: WebhookBatchSize})
				if err != nil {
					return err
				}
				for _, webhook := range webhooks {
					eventID := fmt.Sprintf("deletion-%d", row.ID)
					event.Data.DeletionEventID = eventID
					task, err := newInvokeTask(InvokeArgs{
						Event: event.Event, EventID: eventID, WebhookID: webhook.ID, ImageID: row.ImageID,
						Version: event.Version, EventType: db.WebhookDeletionEvent, Initiator: event.Initiator,
					}, event.Data)
					if err != nil {
						return err
					}
					_, err = enqueue(ctx, task, asynq.TaskID(fmt.Sprintf("%s-webhook-%d", eventID, webhook.ID)), asynq.Retention(24*time.Hour))
					if err != nil && !errors.Is(err, asynq.ErrTaskIDConflict) {
						return err
					}
				}
				if len(webhooks) == WebhookBatchSize {
					if err := q.AdvanceDeletionOutbox(ctx, sqlc.AdvanceDeletionOutboxParams{ID: row.ID, AfterWebhookID: webhooks[len(webhooks)-1].ID}); err != nil {
						return err
					}
					continue
				}
			}
			if err := q.CompleteDeletionOutbox(ctx, row.ID); err != nil {
				return err
			}
		}
		return nil
	})
}

func HandleDeletionOutboxTask(ctx context.Context, _ *asynq.Task) error {
	return dispatchDeletionOutbox(ctx, db.Transaction, client.EnqueueContext)
}
