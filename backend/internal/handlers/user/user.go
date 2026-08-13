package user

import (
	"errors"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func GetSelf(c *gin.Context) {
	userID := c.GetInt64("UserID")

	ctx := c.Request.Context()

	user, err := db.Query().GetUser(ctx, userID)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting user")
		return
	}

	utils.SuccessResponse(c, user)
}

type GetUserResponse struct {
	Versions int64 `json:"versions"`
	Images   int64 `json:"images"`
	sqlc.GetOtherRow
}

func GetUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid user id")
		return
	}

	ctx := c.Request.Context()
	user, err := db.Query().GetOther(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.ErrorResponse(c, 404, "User not found")
		} else {
			utils.ErrorResponse(c, 500, "Failed when collecting user")
		}
		return
	}

	ver, err := db.Query().CountVersionsByUser(ctx, userID)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when counting versions")
		return
	}

	img, err := db.Query().CountImagesByUser(ctx, userID)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when counting images")
		return
	}

	utils.SuccessResponse(c, GetUserResponse{
		GetOtherRow: user,
		Versions:    ver,
		Images:      img,
	})
}

func GetUserContribution(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid user id")
		return
	}

	ctx := c.Request.Context()

	contribution, err := db.Query().GetUserContribution(ctx, userID)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting contribution")
		return
	}

	utils.SuccessResponse(c, contribution)
}

type ListImagesByUserResponse struct {
	Total  int64                     `json:"total"`
	Images []sqlc.GetImagesByUserRow `json:"images"`
}

func ListImagesByUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid user id")
		return
	}

	offset, limit := utils.Pagination(c)

	ctx := c.Request.Context()

	img, err := db.Query().GetImagesByUser(ctx, sqlc.GetImagesByUserParams{
		UserID: userID,
		Offset: int32(offset),
		Limit:  int32(limit),
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when getting image")
		return
	}

	total, err := db.Query().CountImagesByUser(ctx, userID)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when counting images")
		return
	}

	utils.SuccessResponse(c, ListImagesByUserResponse{
		Total:  total,
		Images: img,
	})
}

type UpdatePasswordPayload struct {
	Password *string `json:"password"`
}

func UpdatePassword(c *gin.Context) {

}
