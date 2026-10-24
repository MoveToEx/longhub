package queue_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"github.com/hibiken/asynq"
	"long/internal/config"
	"long/internal/db"
	"long/internal/queue"
	"long/internal/sqlc"
	"long/internal/testutil"
	"long/internal/webhook"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeletionDeliveryAfterRestorationPreservesSnapshotAndRetries(t *testing.T) {
	pool, transact := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	db.Init(ctx, pool)
	t.Cleanup(func() { db.Init(ctx, nil) })
	if _, err := pool.Exec(ctx, `INSERT INTO webhook(user_id,label,endpoint,secret,event_types) VALUES(2,'delete','https://example.com/webhook','secret',4)`); err != nil {
		t.Fatal(err)
	}
	q := sqlc.New(pool)
	id, err := q.CreateDeletionRequest(ctx, sqlc.CreateDeletionRequestParams{ImageID: 1, UserID: 2, Reason: "duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	if err := moderateDeletion(ctx, id, "ACCEPT", "active"); err != nil {
		t.Fatal(err)
	}
	var invocation *asynq.Task
	if err := queue.DispatchDeletionOutboxForTest(ctx, transact, func(_ context.Context, task *asynq.Task, _ ...asynq.Option) (*asynq.TaskInfo, error) {
		if task.Type() == queue.TypeInvoke {
			invocation = task
		}
		return &asynq.TaskInfo{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if invocation == nil {
		t.Fatal("approval did not dispatch webhook")
	}
	if err := moderateDeletion(ctx, id, "REVOKE", "approved"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE version SET text='later text',version=2 WHERE id=1; UPDATE public.user SET username='renamed admin' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request queue.WorkerRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		var payload webhook.Payload
		if err := json.Unmarshal([]byte(request.Body), &payload); err != nil {
			t.Error(err)
		}
		if payload.Event != "delete" || payload.Data.Text != "original text" || payload.Initiator == nil || payload.Initiator.Username != "admin" || payload.Data.DeletionEventID == "" {
			t.Errorf("snapshot changed: %+v", payload)
		}
		calls++
		if calls == 1 {
			w.WriteHeader(503)
		} else {
			w.WriteHeader(200)
		}
	}))
	defer server.Close()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	previous := config.GetConfig().Webhook
	config.GetConfig().Webhook = config.WebhookConfig{PrivateKey: private, Endpoint: server.URL}
	t.Cleanup(func() { config.GetConfig().Webhook = previous })
	if err := queue.HandleInvokeTask(ctx, invocation); err != nil {
		t.Fatal(err)
	}
	var failures int
	if err := pool.QueryRow(ctx, "SELECT failure_count FROM webhook WHERE id=1").Scan(&failures); err != nil || failures != 1 {
		t.Fatal(failures, err)
	}
	if err := queue.HandleInvokeTask(ctx, invocation); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT failure_count FROM webhook WHERE id=1").Scan(&failures); err != nil || failures != 0 || calls != 2 {
		t.Fatal(failures, calls, err)
	}
}

func TestRestorationNotificationRefreshesCurrentStateWithoutNewVersion(t *testing.T) {
	pool, transact := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	db.Init(ctx, pool)
	t.Cleanup(func() { db.Init(ctx, nil) })
	if _, err := pool.Exec(ctx, `INSERT INTO webhook(user_id,label,endpoint,secret,event_types)
VALUES(2,'lifecycle','https://example.com/webhook','secret',4),
      (2,'updates only','https://example.com/updates','secret',2)`); err != nil {
		t.Fatal(err)
	}
	q := sqlc.New(pool)
	id, err := q.CreateDeletionRequest(ctx, sqlc.CreateDeletionRequestParams{ImageID: 1, UserID: 2, Reason: "duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	moderate := func(action string, expected sqlc.DeletionStatus) {
		t.Helper()
		if err := moderateDeletion(ctx, id, action, expected); err != nil {
			t.Fatal(err)
		}
	}
	collect := func() []*asynq.Task {
		t.Helper()
		tasks := []*asynq.Task{}
		if err := queue.DispatchDeletionOutboxForTest(ctx, transact, func(_ context.Context, task *asynq.Task, _ ...asynq.Option) (*asynq.TaskInfo, error) {
			if task.Type() == queue.TypeInvoke {
				var args queue.InvokeArgs
				if err := json.Unmarshal(task.Payload(), &args); err != nil {
					t.Fatal(err)
				}
				if args.WebhookID != 1 || args.EventType != db.WebhookDeletionEvent || args.Version != 1 {
					t.Fatalf("restoration changed version or subscription: %+v", args)
				}
				tasks = append(tasks, task)
			}
			return &asynq.TaskInfo{}, nil
		}); err != nil {
			t.Fatal(err)
		}
		return tasks
	}
	moderate("ACCEPT", "active")
	moderate("REVOKE", "approved")
	tasks := collect()
	if len(tasks) != 2 {
		t.Fatalf("expected delete and restore notifications, got %d", len(tasks))
	}
	// Deliver the old restoration after a second deletion of the same version.
	moderate("ACCEPT", "revoked")
	present := true
	names := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request queue.WorkerRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		var payload webhook.Payload
		if err := json.Unmarshal([]byte(request.Body), &payload); err != nil {
			t.Error(err)
		}
		names[payload.Event]++
		// A sync receiver refreshes canonical state for either event name.
		image, err := db.Query().GetImageDetails(ctx, payload.Data.ImageID)
		if err != nil {
			t.Error(err)
		}
		present = !image.DeletedAt.Valid && image.CurrentVersionID.Valid
	}))
	defer server.Close()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	previous := config.GetConfig().Webhook
	config.GetConfig().Webhook = config.WebhookConfig{PrivateKey: private, Endpoint: server.URL}
	t.Cleanup(func() { config.GetConfig().Webhook = previous })
	for i := len(tasks) - 1; i >= 0; i-- {
		if err := queue.HandleInvokeTask(ctx, tasks[i]); err != nil {
			t.Fatal(err)
		}
	}
	if present {
		t.Fatal("late restoration resurrected a currently deleted image")
	}
	moderate("REVOKE", "approved")
	for _, task := range collect() {
		if err := queue.HandleInvokeTask(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	if !present || names["delete"] != 2 || names["restore"] != 2 {
		t.Fatal("replica did not converge on restored state", present, names)
	}
}
