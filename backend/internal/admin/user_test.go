package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/testutil"
)

const updateUserMutation = `mutation($id:ID!,$input:UpdateUserInput!){updateUser(id:$id,input:$input){id username email permission}}`

func TestUpdateUserGraphQLValidatesBeforeWriting(t *testing.T) {
	pool, _ := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	db.Init(ctx, pool)
	t.Cleanup(func() { db.Init(ctx, nil) })
	q := sqlc.New(pool)
	if err := q.SetUserPermission(ctx, sqlc.SetUserPermissionParams{ID: 3, Permission: db.PermissionRoot | db.PermissionAdmin}); err != nil {
		t.Fatal(err)
	}
	before, err := q.GetUser(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	rootBefore, err := q.GetUser(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, id, email                   string
		actor, actorPermission, requested int64
		password                          *string
		message                           string
	}{
		{name: "unauthenticated", id: "2", email: "valid@example.com", message: "Authentication required"},
		{name: "invalid id", id: "bad", email: "valid@example.com", actor: 1, actorPermission: 8, message: "Invalid user id"},
		{name: "zero id", id: "0", email: "valid@example.com", actor: 1, actorPermission: 8, message: "Invalid user id"},
		{name: "negative id", id: "-1", email: "valid@example.com", actor: 1, actorPermission: 8, message: "Invalid user id"},
		{name: "invalid email", id: "2", email: "not-an-email", actor: 1, actorPermission: 8, message: "valid email"},
		{name: "display name", id: "2", email: "Name <valid@example.com>", actor: 1, actorPermission: 8, message: "valid email"},
		{name: "long email", id: "2", email: strings.Repeat("a", 250) + "@x.com", actor: 1, actorPermission: 8, message: "valid email"},
		{name: "negative permission", id: "2", email: "valid@example.com", actor: 1, actorPermission: 8, requested: -1, message: "R cannot be changed"},
		{name: "admin cannot grant A", id: "2", email: "valid@example.com", actor: 1, actorPermission: 8, requested: 8, message: "Only Root"},
		{name: "admin cannot grant R", id: "2", email: "valid@example.com", actor: 1, actorPermission: 8, requested: 16, message: "R cannot be changed"},
		{name: "root cannot grant R", id: "2", email: "valid@example.com", actor: 1, actorPermission: 16, requested: 16, message: "R cannot be changed"},
		{name: "unknown permission", id: "2", email: "valid@example.com", actor: 1, actorPermission: 16, requested: 128, message: "R cannot be changed"},
		{name: "oversized password", id: "2", email: "valid@example.com", actor: 1, actorPermission: 8, password: new(strings.Repeat("x", 73)), message: "72 bytes"},
		{name: "multibyte password", id: "2", email: "valid@example.com", actor: 1, actorPermission: 8, password: new(strings.Repeat("密", 25)), message: "72 bytes"},
		{name: "ordinary admin editing Root", id: "3", email: "valid@example.com", actor: 1, actorPermission: 8, message: "not found or access denied"},
		{name: "missing user", id: "999", email: "valid@example.com", actor: 1, actorPermission: 8, message: "not found or access denied"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := map[string]any{"email": tc.email, "permission": tc.requested}
			if tc.password != nil {
				input["password"] = *tc.password
			}
			result := invokeGraphQL(t, tc.actor, tc.actorPermission, updateUserMutation, map[string]any{"id": tc.id, "input": input})
			if len(result.Errors) != 1 || !strings.Contains(result.Errors[0].Message, tc.message) {
				t.Fatalf("unexpected result: %+v", result)
			}
			after, err := q.GetUser(ctx, 2)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("invalid update changed the target", err)
			}
			rootAfter, err := q.GetUser(ctx, 3)
			if err != nil || !reflect.DeepEqual(rootBefore, rootAfter) {
				t.Fatal("invalid update changed the Root target", err)
			}
		})
	}
}

func TestUpdateUserGraphQLHashesPasswordsAndUsesActorPermissions(t *testing.T) {
	pool, _ := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	db.Init(ctx, pool)
	t.Cleanup(func() { db.Init(ctx, nil) })
	q := sqlc.New(pool)
	password := strings.Repeat("密", 24) // Exactly 72 UTF-8 bytes.
	mutate := func(actorPermission int64, id string, input map[string]any) {
		t.Helper()
		result := invokeGraphQL(t, 1, actorPermission, updateUserMutation, map[string]any{"id": id, "input": input})
		if len(result.Errors) != 0 {
			t.Fatal(result.Errors)
		}
		var updated struct {
			ID, Username, Email string
			Permission          int64
		}
		if err := json.Unmarshal(result.Data["updateUser"], &updated); err != nil || updated.ID != id {
			t.Fatal(updated, err)
		}
	}
	mutate(8, "2", map[string]any{"email": "  changed@example.com  ", "permission": 7, "password": password})
	user, err := q.GetUser(ctx, 2)
	if err != nil || user.Email != "changed@example.com" || user.Permission != 7 || user.Username != "reporter" {
		t.Fatal(user, err)
	}
	if bytes.Equal(user.PasswordHash, []byte(password)) || bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)) != nil {
		t.Fatal("password was not bcrypt-hashed")
	}
	hash := append([]byte(nil), user.PasswordHash...)
	for _, input := range []map[string]any{
		{"email": "changed@example.com", "permission": 3},
		{"email": "changed@example.com", "permission": 3, "password": ""},
	} {
		mutate(8, "2", input)
		user, err := q.GetUser(ctx, 2)
		if err != nil || !bytes.Equal(hash, user.PasswordHash) {
			t.Fatal("omitted/empty password changed the hash", err)
		}
	}
	mutate(16, "2", map[string]any{"email": "changed@example.com", "permission": 11})
	user, err = q.GetUser(ctx, 2)
	if err != nil || user.Permission != 11 {
		t.Fatal("Root without A could not grant A", user, err)
	}
	if err := q.SetUserPermission(ctx, sqlc.SetUserPermissionParams{ID: 3, Permission: 24}); err != nil {
		t.Fatal(err)
	}
	mutate(16, "3", map[string]any{"email": "root@example.com", "permission": 7})
	user, err = q.GetUser(ctx, 3)
	if err != nil || user.Permission != 23 {
		t.Fatal("Root update did not preserve R while changing A/C/E/D", user, err)
	}
}
