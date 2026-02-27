package handlers

import (
	"context"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/utils"

	"github.com/gin-gonic/gin"
)

type AutocompletePayload struct {
	Prefix string `form:"prefix"`
}

type AutocompleteResponse struct {
	Tags []sqlc.Tag `json:"tags"`
}

func TagAutocomplete(c *gin.Context) {
	var payload AutocompletePayload

	if err := c.ShouldBindQuery(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request: %v", err)
		return
	}

	if len(payload.Prefix) == 0 {
		utils.ErrorResponse(c, 400, "Invalid prefix")
		return
	}

	ctx := context.Background()

	tags, err := db.Query().GetTagsWithPrefix(ctx, sqlc.GetTagsWithPrefixParams{
		Prefix: payload.Prefix,
		Limit:  24,
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting tags: %v", err)
		return
	}

	utils.SuccessResponse(c, AutocompleteResponse{
		Tags: tags,
	})
}

func GetRecommendedTags(c *gin.Context) {
	userID := c.GetInt64("UserID")

	ctx := context.Background()

	tags, err := db.Query().GetFavoriteTags(ctx, sqlc.GetFavoriteTagsParams{
		UserID: userID,
		Limit:  4,
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting tags: %v", err)
		return
	}

	utils.SuccessResponse(c, tags)
}

type RandomImagePayload struct {
	Limit int32 `form:"limit"`
}

func GetRandomImagesByTag(c *gin.Context) {
	ctx := context.Background()

	var payload RandomImagePayload

	if err := c.ShouldBindQuery(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid payload")
		return
	}

	if payload.Limit < 0 || payload.Limit > 24 {
		utils.ErrorResponse(c, 400, "Limit exceeds")
		return
	}

	tag := c.Param("name")

	if len(tag) == 0 {
		utils.ErrorResponse(c, 400, "Invalid tag name")
		return
	}

	img, err := db.Query().GetRandomImage(ctx, sqlc.GetRandomImageParams{
		Name:  tag,
		Limit: payload.Limit,
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting images")
		return
	}

	utils.SuccessResponse(c, img)
}
