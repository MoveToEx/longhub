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

func TestDeletionDeliveryPreservesSnapshotAndRecordsHTTPResults(t *testing.T) {
	pool, _ := testutil.Database(t)
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
	var dispatch, invocation *asynq.Task
	queue.SetEnqueueForTest(t, func(_ context.Context, task *asynq.Task, _ ...asynq.Option) (*asynq.TaskInfo, error) {
		switch task.Type() {
		case queue.TypeDispatch:
			dispatch = task
		case queue.TypeInvoke:
			invocation = task
		}
		return &asynq.TaskInfo{}, nil
	})
	if err := moderateDeletion(ctx, id, "ACCEPT", "active"); err != nil {
		t.Fatal(err)
	}
	if dispatch == nil {
		t.Fatal("approval did not enqueue webhook dispatch")
	}
	if _, err := pool.Exec(ctx, "UPDATE version SET text='later text',version=2 WHERE id=1; UPDATE public.user SET username='renamed admin' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if err := queue.HandleDispatchTask(ctx, dispatch); err != nil {
		t.Fatal(err)
	}
	if invocation == nil {
		t.Fatal("deletion did not dispatch webhook")
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
