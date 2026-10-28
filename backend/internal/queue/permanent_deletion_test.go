package queue

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hibiken/asynq"
	"long/internal/db"
)

func TestObsoleteInvocationIsDiscarded(t *testing.T) {
	// These queued events must stop before touching the database or delivery counters.
	for _, event := range []string{"restore", "unknown"} {
		payload, err := json.Marshal(InvokeArgs{Event: event, EventType: db.WebhookDeletionEvent})
		if err != nil {
			t.Fatal(err)
		}
		if err := HandleInvokeTask(context.Background(), asynq.NewTask(TypeInvoke, payload)); err != nil {
			t.Fatal(err)
		}
	}
}
