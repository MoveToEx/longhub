package middleware

import (
	"github.com/gin-gonic/gin"
)

func Header(name string, value string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header(name, value)
		c.Next()
	}
}
