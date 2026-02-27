package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func Pagination(c *gin.Context) (offset, limit int) {
	soffset := c.Query("offset")

	if len(soffset) == 0 {
		offset = 0
	}

	offset, err := strconv.Atoi(soffset)

	if err != nil {
		offset = 0
	}

	slimit := c.Query("limit")

	if len(slimit) == 0 {
		limit = 24
	}

	limit, err = strconv.Atoi(slimit)

	if err != nil {
		limit = 24
	}

	if limit > 24 {
		limit = 24
	}

	return
}
