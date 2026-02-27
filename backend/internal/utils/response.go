package utils

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Error any `json:"error"`
	Data  any `json:"data,omitempty"`
}

func ErrorResponse(c *gin.Context, status int, format string, args ...any) {
	c.JSON(status, Response{
		Error: fmt.Sprintf(format, args...),
	})
}

func SuccessResponse(c *gin.Context, data any) {
	if data == nil {
		c.JSON(204, nil)
	} else {
		c.JSON(200, Response{
			Error: nil,
			Data:  data,
		})
	}
}
