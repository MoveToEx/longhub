package handlers_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"long/internal/config"
	"long/internal/db"
	"long/internal/routes"
	"long/internal/sqlc"
	"long/internal/testutil"
	"long/internal/utils"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRootUserManagementAuthorization(t *testing.T) {
	pool, _ := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	db.Init(ctx, pool)
	t.Cleanup(func() { db.Init(ctx, nil) })
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	previous := config.GetConfig().JWT
	config.GetConfig().JWT = config.JWTConfig{PrivateKey: private, Issuer: "test", Audience: "test", SessionTTL: 1}
	t.Cleanup(func() { config.GetConfig().JWT = previous })
	q := sqlc.New(pool)
	setPermission := func(id, permission int64) {
		t.Helper()
		if err := q.SetUserPermission(ctx, sqlc.SetUserPermissionParams{ID: id, Permission: permission}); err != nil {
			t.Fatal(err)
		}
	}
	setPermission(1, db.PermissionRoot)
	setPermission(2, db.PermissionCreate|db.PermissionEdit)
	setPermission(3, db.PermissionAdmin)
	rootToken, err := utils.NewToken(1)
	if err != nil {
		t.Fatal(err)
	}
	adminToken, err := utils.NewToken(3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO appkey(user_id, permission, key) VALUES (1,8,'admin-scope'),(1,16,'root-scope')`); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	routes.RegisterRoutes(router)
	update := func(authorization, id string, permission int64, password string, status int, expectedError string) {
		t.Helper()
		body, err := json.Marshal(map[string]any{
			"query":     `mutation UpdateUser($id: ID!, $input: UpdateUserInput!) { updateUser(id: $id, input: $input) { id email permission } }`,
			"variables": map[string]any{"id": id, "input": map[string]any{"email": "updated@example.com", "permission": permission, "password": password}},
		})
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("POST", "/admin/graphql", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authorization)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != status {
			t.Fatalf("status %d, want %d: %s", response.Code, status, response.Body)
		}
		if expectedError != "" {
			if !strings.Contains(response.Body.String(), expectedError) {
				t.Fatal(response.Body.String())
			}
		} else if strings.Contains(response.Body.String(), `"errors"`) {
			t.Fatal(response.Body.String())
		}
	}
	assertPermission := func(id, expected int64) {
		t.Helper()
		user, err := q.GetUser(ctx, id)
		if err != nil || user.Permission != expected {
			t.Fatalf("user %d permission %d, want %d: %v", id, user.Permission, expected, err)
		}
	}

	update("Session "+adminToken, "2", 11, "", 200, "Only Root users")
	assertPermission(2, 3)
	update("Session "+rootToken, "2", 11, "", 200, "")
	assertPermission(2, 11)
	update("Session "+adminToken, "2", 0, "", 200, "")
	assertPermission(2, 8)
	update("Session "+rootToken, "2", 0, "", 200, "")
	assertPermission(2, 0)
	update("Session "+rootToken, "2", 16, "", 200, "R cannot be changed")
	assertPermission(2, 0)

	before, err := q.GetUser(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	update("Session "+adminToken, "1", 7, "replacement-password", 200, "access denied")
	after, err := q.GetUser(ctx, 1)
	if err != nil || after.Email != before.Email || after.Permission != before.Permission || !bytes.Equal(after.PasswordHash, before.PasswordHash) {
		t.Fatalf("ordinary admin modified Root credentials: %v", err)
	}

	setPermission(1, db.PermissionRoot|db.PermissionAdmin)
	update("Key admin-scope", "2", 8, "", 200, "Only Root users")
	assertPermission(2, 0)
	update("Key root-scope", "2", 8, "", 200, "")
	assertPermission(2, 8)
	setPermission(1, db.PermissionAdmin)
	update("Session "+rootToken, "2", 8, "", 200, "Only Root users")
	update("Key root-scope", "2", 8, "", 403, "Admin permission required")
}
