package queue

import (
	"encoding/json"
	"long/internal/db"
	"long/internal/webhook"
	"reflect"
	"testing"
)

func TestInvokeTaskPreservesUpdateChanges(t *testing.T) {
	data := webhook.PreviewData("update")
	task, err := NewInvokeTask(1, data.ImageID, 2, db.WebhookUpdateEvent, data, webhook.PreviewInitiator("update"))
	if err != nil {
		t.Fatal(err)
	}
	var args InvokeArgs
	if err := json.Unmarshal(task.Payload(), &args); err != nil {
		t.Fatal(err)
	}
	body, err := decodeInvokeBody(args.Body)
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := buildInvocationBody(args.EventType, body, `{"old": {{json .Data.Changes.text.Old}}, "new": {{json .Text}}}`, args.Initiator)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Old string `json:"old"`
		New string `json:"new"`
	}
	if err := json.Unmarshal(rendered, &result); err != nil {
		t.Fatal(err)
	}
	if result.Old != "Original image caption" || result.New != data.Text {
		t.Fatalf("unexpected rendered changes: %+v", result)
	}
}

func TestDefaultInvocationSupportsLegacyQueuedBodies(t *testing.T) {
	data := `{"imageID":42,"text":"caption","rating":"none","tags":[],"imageURL":"https://example.com/image.jpg","createdAt":"2026-01-01T12:00:00Z"}`
	legacy, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []json.RawMessage{json.RawMessage(data), legacy} {
		body, err := decodeInvokeBody(raw)
		if err != nil {
			t.Fatal(err)
		}
		result, err := buildInvocationBody(db.WebhookCreationEvent, body, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		if string(result) != `{"event":"create","data":`+data+`}` {
			t.Fatalf("unexpected default payload: %s", result)
		}
	}
}

func TestInvokeTaskPreservesUserIdentities(t *testing.T) {
	for _, event := range []struct {
		name   string
		typeID int64
	}{
		{"create", db.WebhookCreationEvent},
		{"update", db.WebhookUpdateEvent},
		{"delete", db.WebhookDeletionEvent},
	} {
		t.Run(event.name, func(t *testing.T) {
			data := webhook.PreviewData(event.name)
			initiator := webhook.PreviewInitiator(event.name)
			wantUploader, wantInitiator := *data.Uploader, *initiator
			task, err := NewInvokeTask(1, data.ImageID, 2, event.typeID, data, initiator)
			if err != nil {
				t.Fatal(err)
			}
			data.Uploader.Username = "changed owner name"
			initiator.Username = "changed actor name"
			var args InvokeArgs
			if err := json.Unmarshal(task.Payload(), &args); err != nil {
				t.Fatal(err)
			}
			body, err := decodeInvokeBody(args.Body)
			if err != nil {
				t.Fatal(err)
			}
			for _, bodyTemplate := range []string{"", `{"event": {{json .Event}}, "data": {{json .Data}}, "initiator": {{json .Initiator}}}`} {
				rendered, err := buildInvocationBody(args.EventType, body, bodyTemplate, args.Initiator)
				if err != nil {
					t.Fatal(err)
				}
				var result webhook.Payload
				if err := json.Unmarshal(rendered, &result); err != nil {
					t.Fatal(err)
				}
				if result.Event != event.name || !reflect.DeepEqual(result.Data.Uploader, &wantUploader) || !reflect.DeepEqual(result.Initiator, &wantInitiator) {
					t.Fatalf("queued identities changed: %+v", result)
				}
			}
		})
	}
}
