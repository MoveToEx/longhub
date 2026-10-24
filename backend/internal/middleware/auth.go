package middleware

import (
	"context"
	"errors"
	"fmt"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/utils"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func fail(c *gin.Context, status int, msg string, strict bool) {
	if strict == false {
		c.Next()
	} else {
		utils.ErrorResponse(c, status, "%s", msg)
		c.Abort()
	}
}

type authQueries interface {
	GetUser(context.Context, int64) (sqlc.User, error)
	GetUserByAppKey(context.Context, string) (sqlc.GetUserByAppKeyRow, error)
	UpdateAppKeyUsedDate(context.Context, int64) error
}

func Auth(strict bool) gin.HandlerFunc {
	return auth(strict, db.Query())
}

func auth(strict bool, queries authQueries) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, ok := c.Request.Header["Authorization"]

		if !ok || len(val) < 1 {
			fail(c, 401, "Invalid authorization", strict)
			return
		}

		parts := strings.Fields(val[0])

		if len(parts) != 2 {
			fail(c, 401, "Invalid authorization", strict)
			return
		}

		switch parts[0] {
		case "Session":
			claims, err := utils.ParseToken(parts[1])

			if err != nil {
				fail(c, 401, fmt.Sprintf("Invalid session: %v", err), strict)
				return
			}

			sub, err := strconv.ParseInt(claims.Subject, 10, 64)

			if err != nil || sub <= 0 {
				fail(c, 401, "Malformed sub", strict)
				return
			}

			user, err := queries.GetUser(c.Request.Context(), sub)
			if err != nil {
				status, message := 500, "Unable to check authentication"
				if errors.Is(err, pgx.ErrNoRows) {
					status, message = 401, "Invalid authentication"
				}
				fail(c, status, message, strict)
				return
			}

			c.Set("UserID", sub)
			c.Set("Permission", user.Permission)
			c.Set("AuthorizedVia", "Session")

			c.Next()
		case "Key":
			ctx := c.Request.Context()
			user, err := queries.GetUserByAppKey(ctx, parts[1])

			if err != nil {
				fail(c, 401, "Invalid key", strict)
				return
			}

			keyID := user.KeyID

			go func() {
				queries.UpdateAppKeyUsedDate(context.Background(), keyID)
			}()

			c.Set("UserID", user.ID)
			c.Set("Permission", user.KeyPermission&user.Permission)
			c.Set("AuthorizedVia", "AppKey")

			c.Next()
		default:
			fail(c, 401, "Unrecognized authorization type", strict)
			return
		}
	}
}
