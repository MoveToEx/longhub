package handlers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/testutil"
)

func TestDeletionRESTValidationAndPerUserConflicts(t *testing.T) {
	pool, _ := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	db.Init(ctx, pool)
	t.Cleanup(func() { db.Init(ctx, nil) })
	actor := int64(2)
	router := gin.New()
	router.POST("/image/:id/deletion", func(c *gin.Context) { c.Set("UserID", actor) }, CreateImageDeletionRequest)
	router.GET("/image/:id/deletion", ListImageDeletionRequests)
	request := func(method, path, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("%s %s: status %d, want %d: %s", method, path, rec.Code, want, rec.Body.String())
		}
		return rec
	}
	for _, tc := range []struct {
		id, body string
		status   int
	}{
		{"bad", `{"reason":"valid"}`, 400}, {"0", `{"reason":"valid"}`, 400}, {"-1", `{"reason":"valid"}`, 400},
		{"1", `{`, 400}, {"1", `{}`, 400}, {"1", `{"reason":1}`, 400}, {"1", `{"reason":"   "}`, 400},
		{"1", `{"reason":"` + strings.Repeat("密", 2001) + `"}`, 400},
		{"1", `{"reason":"` + strings.Repeat("x", 65*1024) + `"}`, 400},
		{"999", `{"reason":"valid"}`, 404},
	} {
		request("POST", "/image/"+tc.id+"/deletion", tc.body, tc.status)
	}
	created := request("POST", "/image/1/deletion", `{"reason":"  duplicate  "}`, 201)
	var item struct{ Data PublicDeletionRequest }
	if err := json.Unmarshal(created.Body.Bytes(), &item); err != nil || item.Data.Reason != "duplicate" || item.Data.Status != sqlc.DeletionStatusActive || item.Data.Requester.ID != 2 {
		t.Fatal(created.Body.String(), err)
	}
	request("POST", "/image/1/deletion", `{"reason":"another"}`, 409)
	actor = 3
	request("POST", "/image/1/deletion", `{"reason":"`+strings.Repeat("密", 2000)+`"}`, 201)
	list := request("GET", "/image/1/deletion?offset=1&limit=1", "", 200)
	var history struct {
		Data struct {
			Items []PublicDeletionRequest
			Total int64
		}
	}
	if err := json.Unmarshal(list.Body.Bytes(), &history); err != nil || history.Data.Total != 2 || len(history.Data.Items) != 1 || history.Data.Items[0].ID != item.Data.ID {
		t.Fatal(list.Body.String(), err)
	}
	for _, field := range []string{"email", "permission", "passwordHash", "moderator"} {
		if strings.Contains(list.Body.String(), `"`+field+`":`) {
			t.Fatal("private field in public deletion history", field)
		}
	}
	for _, query := range []string{"offset=-1", "offset=2147483648", "offset=bad", "limit=0", "limit=101", "limit=bad"} {
		request("GET", "/image/1/deletion?"+query, "", 400)
	}
	request("GET", "/image/bad/deletion", "", 400)
	request("GET", "/image/999/deletion", "", 404)
	if _, err := pool.Exec(ctx, "UPDATE image SET deleted_at=NOW() WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	request("POST", "/image/1/deletion", `{"reason":"deleted image"}`, 409)
	request("GET", "/image/1/deletion", "", 200)
	total, err := sqlc.New(pool).CountImageDeletionRequests(ctx, 1)
	if err != nil || total != 2 {
		t.Fatal("invalid/conflicting requests created rows", total, err)
	}
}
