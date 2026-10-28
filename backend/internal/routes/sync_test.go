package routes

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"long/internal/config"
	"long/internal/db"
	"long/internal/testutil"
	"long/internal/utils"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSyncRoutesAuthentication(t *testing.T) {
	pool, _ := testutil.Database(t)
	testutil.Seed(t, pool)
	db.Init(context.Background(), pool)
	t.Cleanup(func() { db.Init(context.Background(), nil) })
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	previous := config.GetConfig().JWT
	config.GetConfig().JWT = config.JWTConfig{PrivateKey: private, Issuer: "test", Audience: "test", SessionTTL: 1}
	t.Cleanup(func() { config.GetConfig().JWT = previous })
	token, err := utils.NewToken(2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO appkey(user_id,permission,key) VALUES(2,0,'sync-read-only')`); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	RegisterRoutes(router)
	for _, path := range []string{"/sync/version", "/sync/full"} {
		for _, auth := range []string{"", "Key invalid", "Key sync-read-only", "Session " + token} {
			req := httptest.NewRequest("GET", path, nil)
			if auth != "" {
				req.Header.Set("Authorization", auth)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if auth == "" || auth == "Key invalid" {
				if rec.Code != http.StatusUnauthorized {
					t.Fatalf("auth %q: status = %d, want 401", auth, rec.Code)
				}
				continue
			}
			if rec.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("sync route can be cached", path)
			}
			if path == "/sync/full" {
				if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/x-ndjson" || !strings.Contains(rec.Body.String(), `"type":"manifest"`) || !strings.Contains(rec.Body.String(), `"type":"end"`) {
					t.Fatal("registered full-sync route did not export", rec.Code, rec.Body.String())
				}
				continue
			}
			var body struct {
				Error *string
				Data  struct{ Timestamp *time.Time }
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || rec.Code != http.StatusOK || body.Error != nil || body.Data.Timestamp == nil {
				t.Fatalf("auth %q: response = %d %s, decode error = %v", auth, rec.Code, rec.Body.String(), err)
			}
		}
	}
}
