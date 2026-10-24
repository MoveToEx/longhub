package queue_test

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
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

func TestWebhookHeadersAreSignedAndRenderingFailuresAreRecorded(t *testing.T) {
	pool, _ := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	db.Init(ctx, pool)
	t.Cleanup(func() { db.Init(ctx, nil) })
	q := sqlc.New(pool)
	hook, err := q.NewWebhook(ctx, sqlc.NewWebhookParams{
		UserID: 2, Label: "headers", Endpoint: "https://example.com/hook", Secret: "test-secret", Active: true,
		EventTypes:      db.WebhookDeletionEvent,
		HeaderTemplates: []byte(`{"Authorization":"Bearer token","X-Event":"{{.Event}}","X-Text":"{{.Text}}"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		signature, err := base64.StdEncoding.DecodeString(r.Header.Get("X-Server-Signature"))
		if err != nil || !ed25519.Verify(public, raw, signature) {
			t.Error("envelope signature does not cover headers")
		}
		var envelope queue.WorkerRequest
		if err := json.Unmarshal(raw, &envelope); err != nil {
			t.Error(err)
		}
		if envelope.Headers["Authorization"] != "Bearer token" || envelope.Headers["X-Event"] != "delete" || envelope.Headers["X-Text"] != "Updated image caption" {
			t.Errorf("missing rendered headers: %+v", envelope.Headers)
		}
		mac := hmac.New(sha256.New, []byte("test-secret"))
		mac.Write([]byte(envelope.Body))
		if envelope.ClientSignature != hex.EncodeToString(mac.Sum(nil)) {
			t.Error("client body signature changed")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	previous := config.GetConfig().Webhook
	config.GetConfig().Webhook = config.WebhookConfig{PrivateKey: private, Endpoint: server.URL}
	t.Cleanup(func() { config.GetConfig().Webhook = previous })
	data := webhook.PreviewData("delete")
	invoke := func() error {
		t.Helper()
		task, err := queue.NewInvokeTask(hook.ID, data.ImageID, 1, db.WebhookDeletionEvent, data, webhook.PreviewInitiator("delete"))
		if err != nil {
			t.Fatal(err)
		}
		return queue.HandleInvokeTask(ctx, task)
	}
	if err := invoke(); err != nil {
		t.Fatal(err)
	}
	data.Text = "caption\r\nInjected: true"
	if err := invoke(); err == nil {
		t.Fatal("accepted header injection from event data")
	}
	stored, err := q.GetWebhook(ctx, hook.ID)
	if err != nil || stored.FailureCount != 1 || calls != 1 {
		t.Fatalf("render failure was not recorded before sending: failures=%d calls=%d err=%v", stored.FailureCount, calls, err)
	}
}
