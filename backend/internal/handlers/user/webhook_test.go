package user

import (
	"context"
	"encoding/json"
	"fmt"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/testutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestHeaderTemplatesValidateSubscribedEventsAndTypes(t *testing.T) {
	headers := map[string]string{"X-Old-Text": `{{.Changes.text.Old}}`}
	if err := validateHeaderTemplates(headers, db.WebhookUpdateEvent); err != nil {
		t.Fatal(err)
	}
	if err := validateHeaderTemplates(headers, db.WebhookCreationEvent|db.WebhookUpdateEvent); err == nil {
		t.Fatal("accepted an unguarded update-only field for creation")
	}
	headers = map[string]string{"X-Event": "{{if eq .Event \"restore\"}}bad\nvalue{{else}}{{.Event}}{{end}}"}
	if err := validateHeaderTemplates(headers, db.WebhookDeletionEvent); err == nil {
		t.Fatal("deletion subscription did not validate restoration headers")
	}
	for _, raw := range []string{`[]`, `"header"`, `{"X-Test":1}`, `{"X-Test":null}`, `{"X-Test":true}`} {
		if _, err := decodeHeaderTemplates(json.RawMessage(raw)); err == nil {
			t.Errorf("accepted invalid templates: %s", raw)
		}
	}
}

func TestWebhookHeaderTemplatesCreateListAndEdit(t *testing.T) {
	pool, _ := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	db.Init(ctx, pool)
	t.Cleanup(func() { db.Init(ctx, nil) })
	q := sqlc.New(pool)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("UserID", int64(2)) })
	router.POST("/user/webhook", CreateWebhook)
	router.GET("/user/webhook", ListWebhooks)
	router.PATCH("/user/webhook/:id", EditWebhook)
	request := func(method, path, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != status {
			t.Fatalf("%s %s: HTTP %d: %s", method, path, rec.Code, rec.Body.String())
		}
		return rec
	}
	request("POST", "/user/webhook", `{"label":"headers","endpoint":"https://example.com/hook","secret":"secret","eventTypes":2,"active":true,"headerTemplates":{"Authorization":"Bearer token","X-Old":"{{.Changes.text.Old}}"}}`, http.StatusCreated)
	hooks, err := q.GetWebhooksByUser(ctx, 2)
	if err != nil || len(hooks) != 1 {
		t.Fatalf("stored hooks: %v, %v", hooks, err)
	}
	path := fmt.Sprintf("/user/webhook/%d", hooks[0].ID)
	want := map[string]string{"Authorization": "Bearer token", "X-Old": "{{.Changes.text.Old}}"}
	var list struct {
		Data []ListWebhooksResponse `json:"data"`
	}
	if err := json.Unmarshal(request("GET", "/user/webhook", "", http.StatusOK).Body.Bytes(), &list); err != nil || len(list.Data) != 1 || !reflect.DeepEqual(list.Data[0].HeaderTemplates, want) {
		t.Fatalf("listed headers: %+v, %v", list, err)
	}
	request("PATCH", path, `{"label":"preserve"}`, http.StatusNoContent)
	stored, err := q.GetWebhook(ctx, hooks[0].ID)
	if err != nil || !reflect.DeepEqual(stored.HeaderTemplates, want) {
		t.Fatalf("omitted headers changed: %+v, %v", stored.HeaderTemplates, err)
	}
	request("PATCH", path, `{"eventTypes":3}`, http.StatusBadRequest)
	request("PATCH", path, `{"headerTemplates":{"X-Signature":"override"}}`, http.StatusBadRequest)
	request("PATCH", path, `{"headerTemplates":{"X-Test":null}}`, http.StatusBadRequest)
	request("POST", "/user/webhook", `{"eventTypes":1,"headerTemplates":{"X-Test":null}}`, http.StatusBadRequest)
	for _, raw := range []string{`{"X-New":"{{.Event}}"}`, `{}`, `{"X-New":"static"}`, `null`} {
		request("PATCH", path, `{"headerTemplates":`+raw+`}`, http.StatusNoContent)
		stored, err := q.GetWebhook(ctx, hooks[0].ID)
		if err != nil {
			t.Fatal(err)
		}
		var expected map[string]string
		if err := json.Unmarshal([]byte(raw), &expected); err != nil {
			t.Fatal(err)
		}
		if len(stored.HeaderTemplates) != len(expected) {
			t.Fatalf("headers were merged: %+v", stored.HeaderTemplates)
		}
		for key, value := range expected {
			if stored.HeaderTemplates[key] != value {
				t.Fatalf("stored headers: %+v", stored.HeaderTemplates)
			}
		}
	}
}

func TestDeletionTemplateAlsoValidatesRestoration(t *testing.T) {
	value := `{{if eq .Event "restore"}}invalid JSON{{else}}{}{{end}}`
	if err := validateBodyTemplate(nullableBodyTemplate(&value), db.WebhookDeletionEvent); err == nil {
		t.Fatal("deletion subscription did not validate restoration")
	}
}

func TestEditWebhookPersistsOmittedAndExplicitSettings(t *testing.T) {
	pool, _ := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	db.Init(ctx, pool)
	t.Cleanup(func() { db.Init(ctx, nil) })
	q := sqlc.New(pool)
	router := gin.New()
	router.PATCH("/user/webhook/:id", func(c *gin.Context) { c.Set("UserID", int64(2)) }, EditWebhook)
	const originalTemplate = `{"event": {{json .Event}}}`
	for _, tc := range []struct {
		name, body, wantTemplate string
		active, wantActive       bool
	}{
		{"preserve active", `{"label":"updated"}`, originalTemplate, true, true},
		{"preserve inactive", `{"label":"updated"}`, originalTemplate, false, false},
		{"disable", `{"label":"updated","active":false}`, originalTemplate, true, false},
		{"enable", `{"label":"updated","active":true}`, originalTemplate, false, true},
		{"clear null", `{"label":"updated","bodyTemplate":null}`, "", true, true},
		{"clear empty", `{"label":"updated","bodyTemplate":""}`, "", true, true},
		{"clear whitespace", `{"label":"updated","bodyTemplate":"  "}`, "", true, true},
		{"replace template", `{"label":"updated","bodyTemplate":"{}"}`, "{}", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hook, err := q.NewWebhook(ctx, sqlc.NewWebhookParams{
				UserID: 2, Label: "original", Endpoint: "https://example.com/hook", Secret: "test-secret",
				EventTypes: db.WebhookUpdateEvent, Active: tc.active,
				BodyTemplate: pgtype.Text{String: originalTemplate, Valid: true},
			})
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/user/webhook/%d", hook.ID), strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusNoContent {
				t.Fatalf("edit webhook: HTTP %d: %s", rec.Code, rec.Body.String())
			}
			stored, err := q.GetWebhook(ctx, hook.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Label != "updated" || stored.Active != tc.wantActive {
				t.Fatalf("stored label=%q active=%v, want updated/%v", stored.Label, stored.Active, tc.wantActive)
			}
			if stored.BodyTemplate.Valid != (tc.wantTemplate != "") || stored.BodyTemplate.String != tc.wantTemplate {
				t.Fatalf("stored template=%+v, want %q", stored.BodyTemplate, tc.wantTemplate)
			}
		})
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
