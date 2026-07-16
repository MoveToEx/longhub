package utils

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Error *string `json:"error"`
	Data  any     `json:"data,omitempty"`
}

func ErrorResponse(c *gin.Context, status int, format string, args ...any) {
	errorMessage := fmt.Sprintf(format, args...)
	c.JSON(status, Response{
		Error: &errorMessage,
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
