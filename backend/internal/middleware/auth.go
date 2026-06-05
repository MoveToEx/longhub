package middleware

import (
	"fmt"
	"long/internal/db"
	"long/internal/utils"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func fail(c *gin.Context, status int, msg string, strict bool) {
	if strict == false {
		c.Next()
	} else {
		utils.ErrorResponse(c, status, "%s", msg)
		c.Abort()
	}
}

func Auth(strict bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, ok := c.Request.Header["Authorization"]

		if !ok || len(val) < 1 {
			fail(c, 401, "Invalid authorization", strict)
			return
		}

		parts := strings.Split(val[0], " ")

		if len(parts) < 2 {
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

			if err != nil {
				fail(c, 500, "Malformed sub", strict)
				return
			}

			c.Set("UserID", sub)
			c.Set("Permission", claims.Permission)
			c.Set("AuthorizedVia", "Session")

			c.Next()
		case "Key":
			ctx := c.Request.Context()
			user, err := db.Query().GetUserByAppKey(ctx, parts[1])

			if err != nil {
				fail(c, 401, "Invalid key", strict)
				return
			}

			keyID := user.KeyID

			go func() {
				db.Query().UpdateAppKeyUsedDate(ctx, keyID)
			}()

			c.Set("UserID", user.ID)
			c.Set("Permission", user.KeyPermission)
			c.Set("AuthorizedVia", "AppKey")

			c.Next()
		default:
			fail(c, 401, "Unrecognized authorization type", strict)
			return
		}
	}
}
