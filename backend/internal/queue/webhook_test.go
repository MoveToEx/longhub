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

func TestDeletionInvocationAndLegacyDeletion(t *testing.T) {
	data, err := json.Marshal(webhook.PreviewData("delete"))
	if err != nil {
		t.Fatal(err)
	}
	for _, template := range []string{"", `{"event": {{json .Event}}, "data": {{json .Data}}}`} {
		for _, event := range []string{"", "delete"} {
			body, err := buildInvocationBody(db.WebhookDeletionEvent, data, template, nil, event)
			if err != nil {
				t.Fatal(err)
			}
			var payload webhook.Payload
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatal(err)
			}
			want := event
			if want == "" {
				want = "delete"
			}
			if payload.Event != want || payload.Data.ImageID != 42 {
				t.Fatalf("unexpected lifecycle payload: %+v", payload)
			}
		}
	}
	if _, err := buildInvocationBody(db.WebhookUpdateEvent, data, "", nil, "delete"); err == nil {
		t.Fatal("deletion escaped its subscription")
	}
}

func TestInvocationHeadersUseOriginalSnapshotAndLifecycleEvent(t *testing.T) {
	data := webhook.PreviewData("delete")
	initiator := webhook.PreviewInitiator("delete")
	task, err := NewInvokeTask(1, data.ImageID, 2, db.WebhookDeletionEvent, data, initiator)
	if err != nil {
		t.Fatal(err)
	}
	data.Text = "later caption"
	initiator.Username = "renamed user"
	var args InvokeArgs
	if err := json.Unmarshal(task.Payload(), &args); err != nil {
		t.Fatal(err)
	}
	legacy, err := json.Marshal(string(args.Body))
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []json.RawMessage{args.Body, legacy} {
		body, err := decodeInvokeBody(raw)
		if err != nil {
			t.Fatal(err)
		}
		for _, event := range []string{"", "delete"} {
			headers, err := buildInvocationHeaders(args.EventType, body, map[string]string{
				"X-Event": `{{.Event}}`, "X-Text": `{{.Text}}`, "X-Actor": `{{.Initiator.Username}}`,
			}, args.Initiator, event)
			if err != nil {
				t.Fatal(err)
			}
			wantEvent := event
			if wantEvent == "" {
				wantEvent = "delete"
			}
			if headers["X-Event"] != wantEvent || headers["X-Text"] != "Updated image caption" || headers["X-Actor"] != "image-editor" {
				t.Fatalf("headers lost event snapshot: %+v", headers)
			}
		}
	}
}
