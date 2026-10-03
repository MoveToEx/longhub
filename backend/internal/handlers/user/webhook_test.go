package user

import (
	"encoding/json"
	"long/internal/db"
	"testing"
)

func TestEditWebhookPayloadPreservesOmittedActive(t *testing.T) {
	var payload EditWebhookPayload
	if err := json.Unmarshal([]byte(`{"label":"updated"}`), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Active != nil {
		t.Fatalf("active should be nil when omitted, got %v", *payload.Active)
	}
}

func TestEditWebhookBodyTemplateOmittedOrCleared(t *testing.T) {
	for _, test := range []struct {
		body    string
		present bool
		valid   bool
	}{
		{`{"label":"updated"}`, false, false},
		{`{"bodyTemplate":null}`, true, false},
		{`{"bodyTemplate":""}`, true, false},
		{`{"bodyTemplate":"  "}`, true, false},
		{`{"bodyTemplate":"{}"}`, true, true},
	} {
		var payload EditWebhookPayload
		if err := json.Unmarshal([]byte(test.body), &payload); err != nil {
			t.Fatal(err)
		}
		if (payload.BodyTemplate != nil) != test.present {
			t.Fatalf("presence for %s: %s", test.body, payload.BodyTemplate)
		}
		if test.present {
			var value *string
			if err := json.Unmarshal(payload.BodyTemplate, &value); err != nil {
				t.Fatal(err)
			}
			if nullableBodyTemplate(value).Valid != test.valid {
				t.Fatalf("unexpected validity for %s", test.body)
			}
		}
	}
}

func TestValidateBodyTemplateForSubscribedEvents(t *testing.T) {
	value := `{"old": {{json .Changes.text.Old}}}`
	if err := validateBodyTemplate(nullableBodyTemplate(&value), db.WebhookUpdateEvent); err != nil {
		t.Fatal(err)
	}
	if err := validateBodyTemplate(nullableBodyTemplate(&value), db.WebhookCreationEvent|db.WebhookUpdateEvent); err == nil {
		t.Fatal("expected create event to reject an unguarded update-only field")
	}
	value = `{"uploader": {{json .Data.Uploader}}, "initiatorID": {{.Initiator.ID}}, "initiatorName": {{json .Initiator.Username}}}`
	if err := validateBodyTemplate(nullableBodyTemplate(&value), db.WebhookCreationEvent|db.WebhookUpdateEvent|db.WebhookDeletionEvent); err != nil {
		t.Fatal(err)
	}
}
