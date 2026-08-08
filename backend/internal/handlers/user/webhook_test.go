package user

import (
	"encoding/json"
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
