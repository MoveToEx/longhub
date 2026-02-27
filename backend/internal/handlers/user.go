package handlers

import (
	"context"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/types"
	"long/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetSelf(c *gin.Context) {
	userID := c.GetInt64("UserID")

	ctx := context.Background()

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

	ctx := context.Background()
	user, err := db.Query().GetOther(ctx, userID)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting user: %v", err)
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

	ctx := context.Background()

	contribution, err := db.Query().GetUserContribution(ctx, userID)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting contribution: %v", err)
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

	ctx := context.Background()

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

func UpdatePreference(c *gin.Context) {
	userID, ok := c.Get("UserID")

	if !ok {
		utils.ErrorResponse(c, 500, "Unexpected error")
		return
	}

	var payload types.Preference

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request body")
		return
	}

	ctx := context.Background()

	result := types.Preference{
		HideNSFW:    new(false),
		HideViolent: new(false),
	}

	if payload.HideNSFW != nil {
		result.HideNSFW = payload.HideNSFW
	}

	if payload.HideViolent != nil {
		result.HideViolent = payload.HideViolent
	}

	if err := db.Query().SetUserPreference(ctx, sqlc.SetUserPreferenceParams{
		ID:         userID.(int64),
		Preference: result,
	}); err != nil {
		utils.ErrorResponse(c, 500, "Failed when updating preference")
	}

	utils.SuccessResponse(c, nil)
}

type UpdatePasswordPayload struct {
	Password *string `json:"password"`
}

func UpdatePassword(c *gin.Context) {

}
