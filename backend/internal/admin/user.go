package admin

import (
	"context"
	"errors"
	"log"
	"long/internal/admin/model"
	"long/internal/db"
	"long/internal/sqlc"
	"net/mail"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"golang.org/x/crypto/bcrypt"
)

func (r *Resolver) updateUser(ctx context.Context, id string, input model.UpdateUserInput) (*model.AdminUser, error) {
	actor, _ := ctx.Value(actorContextKey{}).(int64)
	if actor <= 0 {
		return nil, &gqlerror.Error{Message: "Authentication required", Extensions: map[string]any{"code": "UNAUTHENTICATED"}}
	}
	userID, err := strconv.ParseInt(id, 10, 64)
	if err != nil || userID <= 0 {
		return nil, gqlerror.Errorf("Invalid user id")
	}
	email := strings.TrimSpace(input.Email)
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 255 {
		return nil, gqlerror.Errorf("Enter a valid email address of at most 255 bytes")
	}
	permission, _ := ctx.Value(actorPermissionContextKey{}).(int64)
	editable := db.PermissionCreate | db.PermissionEdit | db.PermissionRequestDeletion
	if permission&db.PermissionRoot != 0 {
		editable |= db.PermissionAdmin
	}
	if input.Permission < 0 || input.Permission & ^editable != 0 {
		return nil, gqlerror.Errorf("Only Root users can edit A; R cannot be changed")
	}
	var hash []byte
	if input.Password != nil && *input.Password != "" {
		if len(*input.Password) > 72 {
			return nil, gqlerror.Errorf("Password must be at most 72 bytes")
		}
		hash, err = bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("admin password hashing failed: %v", err)
			return nil, gqlerror.Errorf("Unable to update user")
		}
	}
	store, ok := r.Store.(UserManagementStore)
	if !ok {
		return nil, gqlerror.Errorf("Unable to update user")
	}
	row, err := store.AdminUpdateUser(ctx, sqlc.AdminUpdateUserParams{
		ID: userID, Email: email, Permission: input.Permission, PasswordHash: hash, EditablePermissions: editable,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, &gqlerror.Error{Message: "User not found or access denied", Extensions: map[string]any{"code": "NOT_FOUND"}}
	}
	if err != nil {
		log.Printf("admin user update failed: %v", err)
		return nil, gqlerror.Errorf("Unable to update user")
	}
	return &model.AdminUser{
		ID: strconv.FormatInt(row.ID, 10), Username: row.Username, Email: row.Email,
		Permission: row.Permission, CreatedAt: row.CreatedAt.Time,
		ActiveImageCount: row.ActiveImageCount, VersionCount: row.VersionCount,
	}, nil
}
