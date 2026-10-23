package handlers_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"long/internal/config"
	"long/internal/db"
	"long/internal/deletion"
	"long/internal/routes"
	"long/internal/sqlc"
	"long/internal/testutil"
	"long/internal/utils"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeletionHTTPAndVisibility(t *testing.T) {
	pool, transact := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	db.Init(ctx, pool)
	t.Cleanup(func() { db.Init(ctx, nil) })
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	prior := config.GetConfig().JWT
	config.GetConfig().JWT = config.JWTConfig{PrivateKey: private, Issuer: "test", Audience: "test", SessionTTL: 1}
	t.Cleanup(func() { config.GetConfig().JWT = prior })
	token, err := utils.NewToken(2, 4)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := utils.NewToken(1, 8)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO appkey(user_id,permission,key) VALUES(2,4,'D-key'),(2,8,'A-key')`); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	routes.RegisterRoutes(router)
	call := func(method, path, body, auth string, want int) string {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("%s %s => %d want %d: %s", method, path, rec.Code, want, rec.Body.String())
		}
		return rec.Body.String()
	}
	call("POST", "/image/1/deletion", `{"reason":"duplicate"}`, "", 401)
	call("POST", "/image/1/deletion", `{"reason":"duplicate"}`, "Session "+admin, 403)
	call("POST", "/image/1/deletion", `{"reason":"duplicate"}`, "Key A-key", 403)
	for _, reason := range []string{"", " \n\t", strings.Repeat("字", 2001)} {
		body, _ := json.Marshal(map[string]string{"reason": reason})
		call("POST", "/image/1/deletion", string(body), "Session "+token, 400)
	}
	call("POST", "/image/1/deletion", `{"reason":12}`, "Session "+token, 400)
	call("POST", "/image/1/deletion", `{"reason":"`+strings.Repeat("a", 70000)+`"}`, "Session "+token, 400)
	call("POST", "/image/999/deletion", `{"reason":"missing"}`, "Session "+token, 404)
	body := call("POST", "/image/1/deletion", `{"reason":"  duplicate original  "}`, "Key D-key", 201)
	var created struct {
		Data struct {
			ID     int64
			Reason string
		}
	}
	if err := json.Unmarshal([]byte(body), &created); err != nil || created.Data.Reason != "duplicate original" {
		t.Fatal(body, err)
	}
	call("POST", "/image/1/deletion", `{"reason":"again"}`, "Session "+token, 409)
	if _, err := pool.Exec(ctx, "UPDATE public.user SET permission=0 WHERE id=2"); err != nil {
		t.Fatal(err)
	}
	call("POST", "/image/2/deletion", `{"reason":"permission revoked"}`, "Session "+token, 403)
	call("POST", "/image/2/deletion", `{"reason":"owner revoked"}`, "Key D-key", 403)
	if _, err := pool.Exec(ctx, "UPDATE public.user SET permission=4 WHERE id=2"); err != nil {
		t.Fatal(err)
	}
	service := deletion.Service{Transact: transact}
	if err := service.Moderate(ctx, created.Data.ID, 1, "ACCEPT", "active"); err != nil {
		t.Fatal(err)
	}
	detail := call("GET", "/image/1", "", "", 200)
	if !strings.Contains(detail, `"reason":"duplicate original"`) || !strings.Contains(detail, `"deletedAt":`) || !strings.Contains(detail, `"text":"original text"`) {
		t.Fatal(detail)
	}
	call("GET", "/image/1/version", "", "", 200)
	history := call("GET", "/image/1/deletion?offset=0&limit=1", "", "", 200)
	if !strings.Contains(history, `"status":"approved"`) || !strings.Contains(history, `"total":1`) {
		t.Fatal(history)
	}
	for _, field := range []string{"email", "permission", "processedBy", "moderator", "password"} {
		if strings.Contains(history, `"`+field+`"`) {
			t.Fatal("public history leaked", field)
		}
	}
	call("GET", "/image/1/deletion?limit=101", "", "", 400)
	call("GET", "/image/999/deletion", "", "", 404)
	call("PATCH", "/image/1", `{"text":"edit","tags":[],"rating":"none"}`, "Session "+admin, 403)
	if _, err := pool.Exec(ctx, "UPDATE public.user SET permission=6 WHERE id=2"); err != nil {
		t.Fatal(err)
	}
	call("PATCH", "/image/1", `{"text":"edit","tags":[],"rating":"none"}`, "Session "+token, 403) // old session has no Edit
	editor, err := utils.NewToken(2, 6)
	if err != nil {
		t.Fatal(err)
	}
	call("PATCH", "/image/1", `{"text":"edit","tags":[],"rating":"none"}`, "Session "+editor, 404)
	call("POST", "/favorite", `{"imageId":1}`, "Session "+admin, 404)
	call("POST", "/image/1/deletion", `{"reason":"already deleted"}`, "Session "+token, 409)
	for _, path := range []string{"/image", "/user/2/image", "/favorite"} {
		got := call("GET", path, "", "Session "+token, 200)
		if strings.Contains(got, `"id":1,`) || strings.Contains(got, `"imageUrl":"https://example.com/test.png"`) {
			t.Fatal("deleted image leaked", path, got)
		}
	}
	if err := service.Moderate(ctx, created.Data.ID, 1, "REVOKE", "approved"); err != nil {
		t.Fatal(err)
	}
	restored := call("GET", "/favorite", "", "Session "+token, 200)
	if !strings.Contains(restored, `"total":1`) || !strings.Contains(restored, "https://example.com/test.png") {
		t.Fatal(restored)
	}
	query := fmt.Sprintf(`{"query":"mutation { moderateDeletionRequest(id: \"%d\", action: ACCEPT, expectedStatus: REVOKED) { id } }"}`, created.Data.ID)
	call("POST", "/admin/graphql", query, "Session "+token, 403)
	if _, err := pool.Exec(ctx, "UPDATE public.user SET permission=0 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	call("POST", "/admin/graphql", query, "Session "+admin, 403)
	if rows, err := sqlc.New(pool).GetImagesByIds(ctx, []int64{1}); err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
}
