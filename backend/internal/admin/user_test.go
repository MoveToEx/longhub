package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"long/internal/db"
	"long/internal/sqlc"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

const updateUserMutation = `mutation UpdateUser($id: ID!, $input: UpdateUserInput!) {
  updateUser(id: $id, input: $input) { id username email permission activeImageCount versionCount }
}`

type userManagementTestStore struct {
	testStore
	updateParams     sqlc.AdminUpdateUserParams
	updateCalls      int
	updateError      error
	targetPermission int64
}

func (s *userManagementTestStore) AdminUpdateUser(_ context.Context, p sqlc.AdminUpdateUserParams) (sqlc.AdminUpdateUserRow, error) {
	s.updateCalls++
	s.updateParams = p
	if s.targetPermission&db.PermissionRoot != 0 && p.EditablePermissions&db.PermissionAdmin == 0 {
		return sqlc.AdminUpdateUserRow{}, pgx.ErrNoRows
	}
	return sqlc.AdminUpdateUserRow{
		ID: p.ID, Username: "target", Email: p.Email,
		Permission: (s.targetPermission & ^p.EditablePermissions) | (p.Permission & p.EditablePermissions),
		CreatedAt:  pgtype.Timestamptz{Time: time.Now(), Valid: true}, ActiveImageCount: 2, VersionCount: 3,
	}, s.updateError
}

func updateUserRequest(t *testing.T, store Store, id string, input map[string]any, actorPermissions ...int64) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]any{"query": updateUserMutation, "variables": map[string]any{"id": id, "input": input}})
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	permission := db.PermissionAdmin
	if len(actorPermissions) > 0 {
		permission = actorPermissions[0]
	}
	router.POST("/admin/graphql", func(c *gin.Context) {
		c.Set("UserID", int64(1))
		c.Set("Permission", permission)
	}, NewHandler(store))
	req := httptest.NewRequest("POST", "/admin/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

func TestUpdateUserHashesPassword(t *testing.T) {
	store := &userManagementTestStore{targetPermission: db.PermissionAdmin}
	response := updateUserRequest(t, store, "9007199254740993", map[string]any{
		"email": " updated@example.com ", "permission": 5, "password": "new-password",
	})
	body := response.Body.String()
	if response.Code != 200 || strings.Contains(body, `"errors"`) {
		t.Fatal(body)
	}
	p := store.updateParams
	if store.updateCalls != 1 || p.ID != 9007199254740993 || p.Email != "updated@example.com" || p.Permission != 5 || p.EditablePermissions != 7 {
		t.Fatalf("unexpected update: %+v", p)
	}
	if err := bcrypt.CompareHashAndPassword(p.PasswordHash, []byte("new-password")); err != nil {
		t.Fatalf("password was not hashed correctly: %v", err)
	}
	for _, expected := range []string{`"id":"9007199254740993"`, `"permission":13`, `"activeImageCount":2`, `"versionCount":3`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("missing %s in %s", expected, body)
		}
	}
	if strings.Contains(body, "new-password") || strings.Contains(body, "passwordHash") {
		t.Fatal("response exposed password data")
	}
}

func TestUpdateUserKeepsOmittedOrEmptyPassword(t *testing.T) {
	for _, password := range []any{nil, ""} {
		store := &userManagementTestStore{}
		input := map[string]any{"email": "user@example.com", "permission": 0}
		if password != nil {
			input["password"] = password
		}
		response := updateUserRequest(t, store, "2", input)
		if strings.Contains(response.Body.String(), `"errors"`) || store.updateCalls != 1 || store.updateParams.PasswordHash != nil {
			t.Fatalf("blank password must remain unchanged: %s", response.Body)
		}
	}
}

func TestUpdateUserRejectsInvalidInputBeforeWriting(t *testing.T) {
	for _, tc := range []struct {
		name, id, email, password string
		permission                int64
	}{
		{name: "grant admin", id: "2", email: "valid@example.com", permission: 8},
		{name: "include admin", id: "2", email: "valid@example.com", permission: 15},
		{name: "grant root", id: "2", email: "valid@example.com", permission: 16},
		{name: "unknown permission", id: "2", email: "valid@example.com", permission: 128},
		{name: "negative permission", id: "2", email: "valid@example.com", permission: -1},
		{name: "invalid email", id: "2", email: "invalid"},
		{name: "display name", id: "2", email: "User <user@example.com>"},
		{name: "long email", id: "2", email: strings.Repeat("a", 250) + "@example.com"},
		{name: "invalid id", id: "invalid", email: "valid@example.com"},
		{name: "zero id", id: "0", email: "valid@example.com"},
		{name: "overflow id", id: "9223372036854775808", email: "valid@example.com"},
		{name: "long password", id: "2", email: "valid@example.com", password: strings.Repeat("x", 73)},
		{name: "multibyte password", id: "2", email: "valid@example.com", password: strings.Repeat("密", 25)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &userManagementTestStore{}
			response := updateUserRequest(t, store, tc.id, map[string]any{"email": tc.email, "permission": tc.permission, "password": tc.password})
			if !strings.Contains(response.Body.String(), `"errors"`) || store.updateCalls != 0 {
				t.Fatalf("invalid input reached database: %s", response.Body)
			}
		})
	}
}

func TestUpdateUserRootPermissions(t *testing.T) {
	for _, tc := range []struct {
		name                                 string
		actor, original, requested, expected int64
		allowed                              bool
	}{
		{"admin cannot grant A", 8, 0, 8, 0, false},
		{"admin preserves A", 8, 8, 7, 15, true},
		{"root grants A", 16, 0, 8, 8, true},
		{"root revokes A", 16, 8, 0, 0, true},
		{"root preserves R", 24, 24, 7, 23, true},
		{"root cannot grant R", 16, 0, 16, 0, false},
		{"root cannot submit R even if already granted", 24, 16, 31, 0, false},
		{"root cannot set unknown bits", 16, 0, 128, 0, false},
		{"scoped admin key cannot grant A", 8, 0, 8, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &userManagementTestStore{targetPermission: tc.original}
			response := updateUserRequest(t, store, "2", map[string]any{
				"email": "target@example.com", "permission": tc.requested,
			}, tc.actor)
			body := response.Body.String()
			if !tc.allowed {
				if !strings.Contains(body, `"errors"`) || store.updateCalls != 0 {
					t.Fatalf("unauthorized permission update reached database: %s", body)
				}
				return
			}
			if strings.Contains(body, `"errors"`) || store.updateCalls != 1 {
				t.Fatal(body)
			}
			var result struct {
				Data struct{ UpdateUser struct{ Permission int64 } }
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Data.UpdateUser.Permission != tc.expected {
				t.Fatalf("permission %d, want %d", result.Data.UpdateUser.Permission, tc.expected)
			}
			mask := int64(7)
			if tc.actor&db.PermissionRoot != 0 {
				mask = 15
			}
			if store.updateParams.EditablePermissions != mask {
				t.Fatalf("editable permissions %d, want %d", store.updateParams.EditablePermissions, mask)
			}
		})
	}
}

func TestAdminCannotResetRootCredentials(t *testing.T) {
	store := &userManagementTestStore{targetPermission: db.PermissionRoot}
	response := updateUserRequest(t, store, "2", map[string]any{
		"email": "changed@example.com", "permission": 7, "password": "replacement-password",
	})
	if !strings.Contains(response.Body.String(), "access denied") {
		t.Fatal(response.Body.String())
	}
}

func TestUpdateUserErrors(t *testing.T) {
	input := map[string]any{"email": "user@example.com", "permission": 0}
	store := &userManagementTestStore{}
	response := request(t, store, updateUserMutation, map[string]any{"id": "2", "input": input})
	if !strings.Contains(response.Body.String(), "Authentication required") || store.updateCalls != 0 {
		t.Fatal(response.Body.String())
	}
	for _, tc := range []struct {
		err     error
		message string
	}{
		{pgx.ErrNoRows, "User not found"},
		{errors.New("private database details"), "Unable to update user"},
	} {
		store := &userManagementTestStore{updateError: tc.err}
		response := updateUserRequest(t, store, "2", input)
		body := response.Body.String()
		if !strings.Contains(body, tc.message) || strings.Contains(body, "private database details") {
			t.Fatal(body)
		}
	}
}
