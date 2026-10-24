package middleware

import (
	"crypto/ed25519"
	"crypto/rand"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"long/internal/config"
	"long/internal/db"
	"long/internal/utils"
	"net/http/httptest"
	"testing"
)

func TestDeletionRequiresCurrentDBit(t *testing.T) {
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	previous := config.GetConfig().JWT
	config.GetConfig().JWT = config.JWTConfig{PrivateKey: key, Issuer: "test", Audience: "test", SessionTTL: 1}
	t.Cleanup(func() { config.GetConfig().JWT = previous })
	token, err := utils.NewToken(1)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, auth string
		owner, key int64
		missing    bool
		want       int
	}{
		{"missing", "", 4, 4, false, 401}, {"D session", "Session " + token, 4, 0, false, 204},
		{"A without D", "Session " + token, 8, 0, false, 403}, {"A plus D", "Session " + token, 12, 0, false, 204},
		{"revoked D", "Session " + token, 3, 0, false, 403}, {"deleted account", "Session " + token, 4, 0, true, 401},
		{"D key", "Key valid-key", 4, 4, false, 204}, {"key lacks D", "Key valid-key", 12, 8, false, 403},
		{"owner lacks D", "Key valid-key", 8, 12, false, 403}, {"invalid key", "Key unknown", 4, 4, false, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &identityStore{permission: tc.owner, keyPermission: tc.key}
			if tc.missing {
				store.userError = pgx.ErrNoRows
			}
			router := gin.New()
			router.POST("/request", auth(true, store), requirePermission(store, db.PermissionRequestDeletion, "D required"), func(c *gin.Context) { c.Status(204) })
			req := httptest.NewRequest("POST", "/request", nil)
			req.Header.Set("Authorization", tc.auth)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatal(rec.Code, rec.Body.String())
			}
		})
	}
	// The same session follows permission changes without issuing a new token.
	store := &identityStore{permission: 4}
	router := gin.New()
	router.POST("/request", auth(true, store), requirePermission(store, 4, "D required"), func(c *gin.Context) { c.Status(204) })
	for _, permission := range []int64{4, 0, 4} {
		store.permission = permission
		req := httptest.NewRequest("POST", "/request", nil)
		req.Header.Set("Authorization", "Session "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		want := 204
		if permission == 0 {
			want = 403
		}
		if rec.Code != want {
			t.Fatal(rec.Code)
		}
	}
}
