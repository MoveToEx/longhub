package queue

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hibiken/asynq"
	"long/internal/config"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/testutil"
	"long/internal/webhook"
)

func webhookPolicyFixture(t *testing.T) (context.Context, *sqlc.Queries, sqlc.Webhook, *asynq.Task) {
	t.Helper()
	pool, _ := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	db.Init(ctx, pool)
	t.Cleanup(func() { db.Init(ctx, nil) })
	q := sqlc.New(pool)
	hook, err := q.NewWebhook(ctx, sqlc.NewWebhookParams{UserID: 2, Label: "policy", Endpoint: "https://example.com/hook", Secret: "secret", EventTypes: db.WebhookDeletionEvent, Active: true})
	if err != nil {
		t.Fatal(err)
	}
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	previous := config.GetConfig().Webhook
	config.GetConfig().Webhook = config.WebhookConfig{PrivateKey: private, Endpoint: "https://gateway.example/invoke"}
	t.Cleanup(func() { config.GetConfig().Webhook = previous })
	data := webhook.PreviewData("delete")
	task, err := NewInvokeTask(hook.ID, data.ImageID, 1, db.WebhookDeletionEvent, data, nil)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, q, hook, task
}

func TestWebhookHTTPFailuresSuspendAndReactivate(t *testing.T) {
	ctx, q, hook, task := webhookPolicyFixture(t)
	var status, calls atomic.Int32
	status.Store(http.StatusServiceUnavailable)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(int(status.Load()))
	}))
	defer server.Close()
	config.GetConfig().Webhook.Endpoint = server.URL
	assertState := func(failures int32, active bool, response int32) {
		t.Helper()
		got, err := q.GetWebhook(ctx, hook.ID)
		if err != nil || got.FailureCount != failures || got.Active != active || !got.LastResponseStatus.Valid || got.LastResponseStatus.Int32 != response {
			t.Fatalf("unexpected webhook state: %+v, %v", got, err)
		}
	}
	invoke := func() {
		t.Helper()
		if err := HandleInvokeTask(ctx, task); err != nil {
			t.Fatalf("recorded HTTP result must complete the Asynq task: %v", err)
		}
	}
	for failures := int32(1); failures <= 2; failures++ {
		invoke()
		assertState(failures, true, 503)
	}
	status.Store(http.StatusOK)
	invoke()
	assertState(0, true, 200)
	status.Store(http.StatusServiceUnavailable)
	for failures := int32(1); failures <= WebhookMaxFailures; failures++ {
		invoke()
		assertState(failures, failures < WebhookMaxFailures, 503)
	}
	before := calls.Load()
	invoke()
	if calls.Load() != before {
		t.Fatal("suspended webhook was delivered")
	}
	assertState(WebhookMaxFailures, false, 503)
	if err := q.UpdateWebhook(ctx, sqlc.UpdateWebhookParams{ID: hook.ID, Label: hook.Label, Endpoint: hook.Endpoint, Secret: hook.Secret, EventTypes: hook.EventTypes, Active: true}); err != nil {
		t.Fatal(err)
	}
	assertState(0, true, 503)
	status.Store(http.StatusOK)
	invoke()
	assertState(0, true, 200)
	if calls.Load() != before+1 {
		t.Fatal("reactivated webhook was not delivered")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestWebhookTransportErrorsAreRetryableUntilSuspended(t *testing.T) {
	ctx, q, hook, task := webhookPolicyFixture(t)
	networkError := errors.New("network unavailable")
	calls := 0
	previous := webhookHTTPClient
	webhookHTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, networkError
	})}
	t.Cleanup(func() { webhookHTTPClient = previous })
	for failures := int32(1); failures <= WebhookMaxFailures; failures++ {
		if err := HandleInvokeTask(ctx, task); !errors.Is(err, networkError) {
			t.Fatalf("transport error must reach Asynq: %v", err)
		}
		got, err := q.GetWebhook(ctx, hook.ID)
		if err != nil || got.FailureCount != failures || got.Active != (failures < WebhookMaxFailures) || got.LastResponseStatus.Valid {
			t.Fatalf("transport failure state: %+v, %v", got, err)
		}
	}
	if err := HandleInvokeTask(ctx, task); err != nil {
		t.Fatal("suspended task should complete without retry", err)
	}
	if calls != WebhookMaxFailures {
		t.Fatal("suspended webhook reached the transport", calls)
	}
}
