package middleware

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"long/internal/config"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/utils"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type identityStore struct {
	permission, keyPermission int64
	userError                 error
}

func (s *identityStore) GetUser(context.Context, int64) (sqlc.User, error) {
	return sqlc.User{ID: 1, Permission: s.permission}, s.userError
}
func (s *identityStore) GetUserByAppKey(_ context.Context, key string) (sqlc.GetUserByAppKeyRow, error) {
	if key != "valid-key" {
		return sqlc.GetUserByAppKeyRow{}, pgx.ErrNoRows
	}
	return sqlc.GetUserByAppKeyRow{ID: 1, KeyID: 1, Permission: s.permission, KeyPermission: s.keyPermission}, nil
}
func (*identityStore) UpdateAppKeyUsedDate(context.Context, int64) error { return nil }

func TestAdminAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	previous := config.GetConfig().JWT
	config.GetConfig().JWT = config.JWTConfig{PrivateKey: private, Issuer: "test", Audience: "test", SessionTTL: 1}
	t.Cleanup(func() { config.GetConfig().JWT = previous })
	token, err := utils.NewToken(1, db.PermissionAdmin)
	if err != nil {
		t.Fatal(err)
	}
	oldToken, err := utils.NewToken(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, auth                string
		permission, keyPermission int64
		userError                 error
		status                    int
	}{
		{name: "missing", status: 401},
		{name: "invalid session", auth: "Session bad-token", status: 401},
		{name: "malformed header", auth: "Session a b", status: 401},
		{name: "unknown scheme", auth: "Bearer token", status: 401},
		{name: "invalid key", auth: "Key unknown", status: 401},
		{name: "ordinary user", auth: "Session " + token, permission: db.PermissionCreate | db.PermissionEdit, status: 403},
		{name: "unknown permission grants no admin access", auth: "Session " + token, permission: 1 << 7, status: 403},
		{name: "admin session", auth: "Session " + token, permission: db.PermissionAdmin, status: 204},
		{name: "revoked session", auth: "Session " + token, permission: 0, status: 403},
		{name: "newly granted session", auth: "Session " + oldToken, permission: db.PermissionAdmin, status: 204},
		{name: "deleted account", auth: "Session " + token, userError: pgx.ErrNoRows, status: 401},
		{name: "admin key", auth: "Key valid-key", permission: db.PermissionAdmin, keyPermission: db.PermissionAdmin, status: 204},
		{name: "key lacks permission", auth: "Key valid-key", permission: db.PermissionAdmin, status: 403},
		{name: "owner permission revoked", auth: "Key valid-key", permission: 0, keyPermission: db.PermissionAdmin, status: 403},
		{name: "database failure", auth: "Session " + token, userError: errors.New("private database details"), status: 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &identityStore{permission: tc.permission, keyPermission: tc.keyPermission, userError: tc.userError}
			router := gin.New()
			router.POST("/admin/graphql", auth(true, store), requireAdmin(store), func(c *gin.Context) { c.Status(http.StatusNoContent) })
			req := httptest.NewRequest(http.MethodPost, "/admin/graphql", nil)
			req.Header.Set("Authorization", tc.auth)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if response.Code != tc.status {
				t.Fatalf("status %d, want %d: %s", response.Code, tc.status, response.Body)
			}
			if strings.Contains(response.Body.String(), "private database details") {
				t.Fatal("database error leaked")
			}
		})
	}
}

func TestAdminRechecksPermissionOnEveryRequest(t *testing.T) {
	store := &identityStore{permission: db.PermissionAdmin}
	router := gin.New()
	router.POST("/admin/graphql", func(c *gin.Context) { c.Set("UserID", int64(1)); c.Set("AuthorizedVia", "Session") }, requireAdmin(store), func(c *gin.Context) { c.Status(204) })
	for _, permission := range []int64{db.PermissionAdmin, 0, db.PermissionAdmin} {
		store.permission = permission
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("POST", "/admin/graphql", nil))
		want := 204
		if permission == 0 {
			want = 403
		}
		if response.Code != want {
			t.Fatalf("permission %d: got %d", permission, response.Code)
		}
	}
}
