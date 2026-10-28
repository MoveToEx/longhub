package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/testutil"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeletionGraphQLIntegration(t *testing.T) {
	pool, _ := testutil.Database(t)
	testutil.Seed(t, pool)
	q := sqlc.New(pool)
	ctx := context.Background()
	db.Init(ctx, pool)
	t.Cleanup(func() { db.Init(ctx, nil) })
	id, err := q.CreateDeletionRequest(ctx, sqlc.CreateDeletionRequestParams{ImageID: 1, UserID: 2, Reason: "duplicate original"})
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(query string, vars any) string {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"query": query, "variables": vars})
		r := gin.New()
		r.POST("/admin/graphql", func(c *gin.Context) { c.Set("UserID", int64(1)) }, GraphQL)
		req := httptest.NewRequest("POST", "/admin/graphql", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != 200 && rec.Code != 422 {
			t.Fatal(rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}
	list := `query($input: DeletionQueryInput){deletionRequests(input:$input){total items{id imageId reason status createdAt processedAt moderator{id username} requester{id username} imageDeletedAt activeDeletionId}}}`
	got := invoke(list, nil)
	if !strings.Contains(got, `"total":1`) || !strings.Contains(got, `"status":"ACTIVE"`) || !strings.Contains(got, `"id":"1"`) {
		t.Fatal(got)
	}
	mutation := `mutation($id:ID!,$action:DeletionAction!,$expectedStatus:DeletionStatus!){moderateDeletionRequest(id:$id,action:$action,expectedStatus:$expectedStatus){id status moderator{id username} imageDeletedAt activeDeletionId}}`
	vars := map[string]any{"id": id, "action": "ACCEPT", "expectedStatus": "ACTIVE"}
	got = invoke(mutation, vars)
	if strings.Contains(got, `"errors"`) || !strings.Contains(got, `"status":"APPROVED"`) || !strings.Contains(got, `"username":"admin"`) {
		t.Fatal(got)
	}
	got = invoke(mutation, vars)
	if !strings.Contains(got, `"code":"CONFLICT"`) {
		t.Fatal(got)
	}
	for _, action := range []string{"ACCEPT", "REJECT", "REOPEN"} {
		got = invoke(mutation, map[string]any{"id": id, "action": action, "expectedStatus": "APPROVED"})
		if !strings.Contains(got, `"code":"CONFLICT"`) {
			t.Fatalf("approved deletion allowed %s: %s", action, got)
		}
	}
	got = invoke(mutation, map[string]any{"id": id, "action": "REVOKE", "expectedStatus": "APPROVED"})
	if !strings.Contains(got, `"errors"`) {
		t.Fatal("removed moderation action was accepted", got)
	}
	image, err := q.LockImageForDeletion(ctx, 1)
	if err != nil || !image.DeletedAt.Valid || image.ActiveDeletionID.Int64 != id {
		t.Fatal("approved image did not stay deleted", image, err)
	}
	got = invoke(list, nil)
	if !strings.Contains(got, `"total":0`) {
		t.Fatal(got)
	}
	got = invoke(list, map[string]any{"input": map[string]any{"status": "ALL", "search": "duplicate", "limit": 1}})
	if !strings.Contains(got, `"total":1`) || !strings.Contains(got, `"status":"APPROVED"`) {
		t.Fatal(got)
	}
	for _, query := range []string{`{deletionRequests{items{requester{email}}}}`, `{deletionRequests{items{moderator{passwordHash}}}}`, `{deletionRequests(input:{limit:101}){total}}`} {
		if got := invoke(query, nil); !strings.Contains(got, `"errors"`) {
			t.Fatal(got)
		}
	}
}
