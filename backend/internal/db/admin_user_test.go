package db

import (
	"bytes"
	"context"
	"errors"
	"long/internal/sqlc"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestAdminUpdateUserPreservesProtectedPermissionsAndPassword(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, sqlDB := timestampTestDatabase(t, ctx, "UTC")
	if err := Migrate(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	q := sqlc.New(pool)
	oldHash := []byte("old-password-hash")
	if err := q.CreateUser(ctx, sqlc.CreateUserParams{Username: "target", Email: "old@example.com", PasswordHash: oldHash}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name                                string
		original, requested, mask, expected int64
	}{
		{"admin preserves A", 8, 7, 7, 15},
		{"admin cannot grant A", 0, 15, 7, 7},
		{"admin cannot revoke A", 15, 0, 7, 8},
		{"root grants A", 3, 11, 15, 11},
		{"root revokes A", 31, 7, 15, 23},
		{"cannot grant R with excessive mask", 0, 31, 31, 15},
		{"cannot revoke R with excessive mask", 16, 0, 31, 16},
		{"preserves unknown bits", 152, 0, 15, 144},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := q.SetUserPermission(ctx, sqlc.SetUserPermissionParams{ID: 1, Permission: tc.original}); err != nil {
				t.Fatal(err)
			}
			row, err := q.AdminUpdateUser(ctx, sqlc.AdminUpdateUserParams{ID: 1, Email: "new@example.com", Permission: tc.requested, EditablePermissions: tc.mask})
			if err != nil {
				t.Fatal(err)
			}
			if row.Permission != tc.expected || row.Email != "new@example.com" || row.Username != "target" {
				t.Fatalf("got %+v, want permission %d", row, tc.expected)
			}
			user, err := q.GetUser(ctx, 1)
			if err != nil || !bytes.Equal(user.PasswordHash, oldHash) {
				t.Fatalf("omitted password changed: %v", err)
			}
		})
	}
	if err := q.SetUserPermission(ctx, sqlc.SetUserPermissionParams{ID: 1, Permission: PermissionRoot | PermissionAdmin}); err != nil {
		t.Fatal(err)
	}
	before, err := q.GetUser(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.AdminUpdateUser(ctx, sqlc.AdminUpdateUserParams{
		ID: 1, Email: "attacker@example.com", PasswordHash: []byte("replacement"), Permission: 7, EditablePermissions: 7,
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("ordinary admin changed Root account: %v", err)
	}
	after, err := q.GetUser(ctx, 1)
	if err != nil || after.Email != before.Email || after.Permission != before.Permission || !bytes.Equal(after.PasswordHash, before.PasswordHash) {
		t.Fatalf("Root account was modified: %+v, %v", after, err)
	}
	newHash := []byte("new-password-hash")
	if _, err := q.AdminUpdateUser(ctx, sqlc.AdminUpdateUserParams{ID: 1, Email: "new@example.com", PasswordHash: newHash, EditablePermissions: 15}); err != nil {
		t.Fatal(err)
	}
	user, err := q.GetUser(ctx, 1)
	if err != nil || !bytes.Equal(user.PasswordHash, newHash) {
		t.Fatalf("password not updated: %v", err)
	}
	if _, err := q.AdminUpdateUser(ctx, sqlc.AdminUpdateUserParams{ID: 999, Email: "missing@example.com"}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("missing user: %v", err)
	}
}
