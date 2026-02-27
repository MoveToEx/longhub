package handlers

import (
	"context"
	"crypto/rand"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetAppKey(c *gin.Context) {
	userID := c.GetInt64("UserID")

	ctx := context.Background()

	keys, err := db.Query().GetAppKeysByUser(ctx, userID)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting keys")
		return
	}

	utils.SuccessResponse(c, keys)
}

type CreateAppKeyPayload struct {
	Label      *string `json:"label"`
	Permission int64   `json:"permission"`
}

type CreateAppKeyResponse struct {
	ID  int64  `json:"id"`
	Key string `json:"key"`
}

func CreateAppKey(c *gin.Context) {
	userID := c.GetInt64("UserID")
	permission := c.GetInt64("Permission")

	var payload CreateAppKeyPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 500, "Invalid parameters")
		return
	}

	if payload.Permission & ^permission != 0 {
		utils.ErrorResponse(c, 400, "Invalid permission: %d, %d", payload.Permission, permission)
		return
	}

	ctx := context.Background()

	key := "long-" + rand.Text()

	var label string

	if payload.Label != nil {
		label = *payload.Label
	} else {
		label = "New key"
	}

	id, err := db.Query().CreateAppKey(ctx, sqlc.CreateAppKeyParams{
		UserID:     userID,
		Permission: payload.Permission,
		Key:        key,
		Label:      label,
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when creating key")
		return
	}

	utils.SuccessResponse(c, CreateAppKeyResponse{
		ID:  id,
		Key: key,
	})
}

type EditAppKeyPayload struct {
	Label      *string `json:"name"`
	Permission *int64  `json:"permission"`
}

type EditAppKeyResponse struct {
	ID  int64  `json:"id"`
	Key string `json:"key"`
}

func EditAppKey(c *gin.Context) {
	userID := c.GetInt64("UserID")
	permission := c.GetInt64("Permission")

	var payload EditAppKeyPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 500, "Invalid parameters")
		return
	}

	if payload.Permission != nil && (*payload.Permission & ^permission != 0) {
		utils.ErrorResponse(c, 400, "Invalid permission")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		utils.ErrorResponse(c, 500, "Invalid id")
		return
	}

	ctx := context.Background()

	key, err := db.Query().GetAppKeyById(ctx, id)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when getting key")
		return
	}

	if key.UserID != userID {
		utils.ErrorResponse(c, 409, "User ID mismatch")
		return
	}

	params := sqlc.UpdateAppKeyParams{
		ID:         id,
		Permission: key.Permission,
		Label:      key.Label,
	}

	if payload.Label != nil {
		params.Label = *payload.Label
	}
	if payload.Permission != nil {
		params.Permission = *payload.Permission
	}

	err = db.Query().UpdateAppKey(ctx, params)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when creating key")
		return
	}

	utils.SuccessResponse(c, nil)
}

func DeleteAppKey(c *gin.Context) {
	userID := c.GetInt64("UserID")

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		utils.ErrorResponse(c, 500, "Invalid id")
		return
	}

	ctx := context.Background()

	key, err := db.Query().GetAppKeyById(ctx, id)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when getting key")
		return
	}

	if key.UserID != userID {
		utils.ErrorResponse(c, 409, "User ID mismatch")
		return
	}

	err = db.Query().DeleteAppKey(c, id)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when deleting key")
		return
	}

	utils.SuccessResponse(c, nil)
}
