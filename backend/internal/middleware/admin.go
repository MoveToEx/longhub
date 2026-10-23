package middleware

import (
	"context"
	"errors"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type adminQueries interface {
	GetUser(context.Context, int64) (sqlc.User, error)
}

// RequireAdmin follows Auth and deliberately ignores stale session permission claims.
func RequireAdmin() gin.HandlerFunc {
	return requireAdmin(db.Query())
}

func requireAdmin(queries adminQueries) gin.HandlerFunc {
	return requirePermission(queries, db.PermissionAdmin, "Admin permission required")
}

func RequireDeletionPermission() gin.HandlerFunc {
	return requirePermission(db.Query(), db.PermissionRequestDeletion, "Request deletion permission required")
}

func requirePermission(queries adminQueries, required int64, message string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		id := c.GetInt64("UserID")
		if id <= 0 {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid authentication")
			c.Abort()
			return
		}
		user, err := queries.GetUser(c.Request.Context(), id)
		if err != nil {
			status, message := http.StatusInternalServerError, "Unable to check admin access"
			if errors.Is(err, pgx.ErrNoRows) {
				status, message = http.StatusUnauthorized, "Invalid authentication"
			}
			utils.ErrorResponse(c, status, "%s", message)
			c.Abort()
			return
		}
		permission := user.Permission
		if c.GetString("AuthorizedVia") == "AppKey" {
			permission &= c.GetInt64("Permission")
		}
		if permission&required != required {
			utils.ErrorResponse(c, http.StatusForbidden, "%s", message)
			c.Abort()
			return
		}
		c.Next()
	}
}
