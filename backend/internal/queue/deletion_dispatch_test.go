package queue_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"long/internal/db"
	"long/internal/queue"
	"long/internal/sqlc"
	"long/internal/testutil"
)

func TestDeletionDispatchRetriesAndPaginatesWithSnapshot(t *testing.T) {
	ctx := context.Background()
	pool, _ := testutil.Database(t)
	testutil.Seed(t, pool)
	db.Init(ctx, pool)
	t.Cleanup(func() { db.Init(ctx, nil) })
	if _, err := pool.Exec(ctx, `
INSERT INTO webhook(user_id,label,endpoint,secret,event_types)
SELECT 2,'delete-'||n,'https://example.com/webhook','secret',4 FROM generate_series(1,102) n;
INSERT INTO webhook(user_id,label,endpoint,secret,event_types,active,failure_count)
VALUES(2,'updates only','https://example.com/webhook','secret',2,true,0),
      (2,'inactive','https://example.com/webhook','secret',4,false,0),
      (2,'suspended','https://example.com/webhook','secret',4,true,3);
`); err != nil {
		t.Fatal(err)
	}
	var dispatches []*asynq.Task
	invocations := map[string]*asynq.Task{}
	offline := errors.New("queue offline")
	failAfterFirst := true
	queue.SetEnqueueForTest(t, func(_ context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
		if task.Type() == queue.TypeDispatch {
			dispatches = append(dispatches, task)
		} else if task.Type() == queue.TypeInvoke {
			var taskID string
			for _, opt := range opts {
				if opt.Type() == asynq.TaskIDOpt {
					taskID = opt.Value().(string)
				}
			}
			if taskID == "" {
				t.Fatal("deletion invocation has no stable task ID")
			}
			if _, exists := invocations[taskID]; exists {
				return nil, asynq.ErrTaskIDConflict
			}
			if len(invocations) == 1 && failAfterFirst {
				failAfterFirst = false
				return nil, offline
			}
			invocations[taskID] = task
		}
		return &asynq.TaskInfo{}, nil
	})
	id, err := sqlc.New(pool).CreateDeletionRequest(ctx, sqlc.CreateDeletionRequestParams{ImageID: 1, UserID: 2, Reason: "duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	if err := moderateDeletion(ctx, id, "ACCEPT", "active"); err != nil {
		t.Fatal(err)
	}
	if len(dispatches) != 1 {
		t.Fatal("approval did not enqueue exactly one dispatch", len(dispatches))
	}
	if _, err := pool.Exec(ctx, "UPDATE version SET text='later text' WHERE id=1; UPDATE public.user SET username='renamed admin' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if err := queue.HandleDispatchTask(ctx, dispatches[0]); !errors.Is(err, offline) {
		t.Fatal("queue failure was not returned for Asynq retry", err)
	}
	if err := queue.HandleDispatchTask(ctx, dispatches[0]); err != nil {
		t.Fatal("dispatch retry failed", err)
	}
	if len(dispatches) != 2 || len(invocations) != queue.WebhookBatchSize {
		t.Fatal("retry duplicated deliveries or lost pagination", len(dispatches), len(invocations))
	}
	if err := queue.HandleDispatchTask(ctx, dispatches[1]); err != nil {
		t.Fatal(err)
	}
	if len(invocations) != 102 {
		t.Fatal("missing subscriptions or delivered to disabled subscriptions", len(invocations))
	}
	var eventID string
	for _, task := range invocations {
		var args queue.InvokeArgs
		if err := json.Unmarshal(task.Payload(), &args); err != nil {
			t.Fatal(err)
		}
		var body queue.RequestBody
		if err := json.Unmarshal(args.Body, &body); err != nil {
			t.Fatal(err)
		}
		if body.Text != "original text" || args.Initiator == nil || args.Initiator.Username != "admin" || args.Version != 1 || args.EventType != db.WebhookDeletionEvent || body.DeletionEventID == "" {
			t.Fatal("dispatch lost the approval snapshot", args, body)
		}
		if eventID == "" {
			eventID = body.DeletionEventID
		} else if eventID != body.DeletionEventID {
			t.Fatal("event identity changed across pages")
		}
	}
}
