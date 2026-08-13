package handlers

import (
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

	ctx := c.Request.Context()

	tags, err := db.Query().GetTagsWithPrefix(ctx, sqlc.GetTagsWithPrefixParams{
		Prefix: payload.Prefix,
		Limit:  24,
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting tags")
		return
	}

	utils.SuccessResponse(c, AutocompleteResponse{
		Tags: tags,
	})
}

type RecommendedTagResponse struct {
	ID     int64                    `json:"id"`
	Name   string                   `json:"name"`
	Count  int64                    `json:"count"`
	Images []sqlc.GetRandomImageRow `json:"images"`
}

func GetRecommendedTags(c *gin.Context) {
	userID := c.GetInt64("UserID")

	ctx := c.Request.Context()

	tags, err := db.Query().GetFavoriteTags(ctx, sqlc.GetFavoriteTagsParams{
		UserID: userID,
		Limit:  4,
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting tags")
		return
	}

	recommendations := make([]RecommendedTagResponse, 0, len(tags))
	for _, tag := range tags {
		images, err := db.Query().GetRandomImage(ctx, sqlc.GetRandomImageParams{
			Name:  tag.Name,
			Limit: 6,
		})
		if err != nil {
			utils.ErrorResponse(c, 500, "Failed when collecting recommended images")
			return
		}

		recommendations = append(recommendations, RecommendedTagResponse{
			ID:     tag.ID,
			Name:   tag.Name,
			Count:  tag.Count,
			Images: images,
		})
	}

	utils.SuccessResponse(c, recommendations)
}
