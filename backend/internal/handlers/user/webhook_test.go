package user

import (
	"context"
	"fmt"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/testutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

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
