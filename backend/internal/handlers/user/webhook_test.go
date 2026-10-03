package user

import (
	"bytes"
	"encoding/json"
	"long/internal/db"
	"long/internal/webhook"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
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

func TestPreviewWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/preview", PreviewWebhook)
	for _, test := range []struct {
		name    string
		payload PreviewWebhookPayload
		status  int
	}{
		{"default", PreviewWebhookPayload{Event: "create"}, http.StatusOK},
		{"update", PreviewWebhookPayload{Event: "update"}, http.StatusOK},
		{"custom", PreviewWebhookPayload{Event: "delete", BodyTemplate: `{"id": {{.ImageID}}, "event": {{json .Event}}}`}, http.StatusOK},
		{"identities", PreviewWebhookPayload{Event: "update", BodyTemplate: `{"uploader": {{json .Data.Uploader}}, "initiator": {{json .Initiator}}}`}, http.StatusOK},
		{"bad syntax", PreviewWebhookPayload{Event: "create", BodyTemplate: "{{"}, http.StatusBadRequest},
		{"bad JSON", PreviewWebhookPayload{Event: "create", BodyTemplate: `{"text": {{.Text}}}`}, http.StatusBadRequest},
		{"bad field", PreviewWebhookPayload{Event: "create", BodyTemplate: `{{json .Unknown}}`}, http.StatusBadRequest},
		{"bad event", PreviewWebhookPayload{Event: "invalid"}, http.StatusBadRequest},
		{"oversized", PreviewWebhookPayload{Event: "create", BodyTemplate: strings.Repeat("x", webhook.MaxTemplateSize+1)}, http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			body, err := json.Marshal(test.payload)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/preview", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.status, response.Body.String())
			}
			if test.status == http.StatusOK {
				var result struct {
					Data struct {
						Body json.RawMessage `json:"body"`
					} `json:"data"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || !json.Valid(result.Data.Body) {
					t.Fatalf("invalid preview response: %s (%v)", response.Body.String(), err)
				}
			}
		})
	}
}
