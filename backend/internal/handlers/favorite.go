package handlers

import (
	"context"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

func GetFavoriteState(c *gin.Context) {
	imageId, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid request")
		return
	}

	ctx := context.Background()
	userID := c.GetInt64("UserID")

	fav, err := db.Query().GetFavoriteState(ctx, sqlc.GetFavoriteStateParams{
		UserID:  userID,
		ImageID: imageId,
	})

	if err != nil {
		utils.SuccessResponse(c, nil)
	} else {
		utils.SuccessResponse(c, fav)
	}
}

type GetFavoritesResponse struct {
	Total  int64                       `json:"total"`
	Images []sqlc.GetFavoriteImagesRow `json:"images"`
}

func GetFavorites(c *gin.Context) {
	offset, limit := utils.Pagination(c)
	userID := c.GetInt64("UserID")
	ctx := context.Background()

	fav, err := db.Query().GetFavoriteImages(ctx, sqlc.GetFavoriteImagesParams{
		Offset: int32(offset),
		Limit:  int32(limit),
		UserID: userID,
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting images")
		return
	}

	total, err := db.Query().CountFavoriteImages(ctx, userID)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when counting images")
		return
	}

	utils.SuccessResponse(c, GetFavoritesResponse{
		Total:  total,
		Images: fav,
	})
}

type SetFavoriteShortcutPayload struct {
	Shortcut *string `json:"shortcut"`
}

func SetFavoriteShortcut(c *gin.Context) {
	imageId, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid request")
		return
	}

	var payload SetFavoriteShortcutPayload

	if err = c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request")
		return
	}

	ctx := context.Background()
	userID := c.GetInt64("UserID")

	shortcut := pgtype.Text{
		Valid: false,
	}

	if payload.Shortcut != nil && len(*payload.Shortcut) != 0 {
		shortcut = pgtype.Text{
			Valid:  true,
			String: *payload.Shortcut,
		}
	}

	err = db.Query().SetFavoriteShortcut(ctx, sqlc.SetFavoriteShortcutParams{
		UserID:   userID,
		ImageID:  imageId,
		Shortcut: shortcut,
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when updating row")
		return
	}

	utils.SuccessResponse(c, nil)
}

type AddFavoritePayload struct {
	ImageID  int64   `json:"imageId"`
	Shortcut *string `json:"shortcut"`
}

func AddFavorite(c *gin.Context) {
	var payload AddFavoritePayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request")
		return
	}

	ctx := context.Background()
	userID := c.GetInt64("UserID")

	var shortcut = pgtype.Text{
		Valid: false,
	}

	if payload.Shortcut != nil {
		shortcut = pgtype.Text{
			Valid:  true,
			String: *payload.Shortcut,
		}
	}

	err := db.Query().AddFavorite(ctx, sqlc.AddFavoriteParams{
		UserID:   userID,
		ImageID:  payload.ImageID,
		Shortcut: shortcut,
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when inserting row")
		return
	}

	utils.SuccessResponse(c, nil)
}

func DeleteFavorite(c *gin.Context) {
	imageId, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid request")
		return
	}

	ctx := context.Background()
	userID := c.GetInt64("UserID")

	err = db.Query().DeleteFavorite(ctx, sqlc.DeleteFavoriteParams{
		UserID:  userID,
		ImageID: imageId,
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when inserting row")
		return
	}

	utils.SuccessResponse(c, nil)
}
