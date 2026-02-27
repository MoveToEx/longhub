package handlers

import (
	"context"
	"long/internal/config"
	"long/internal/constant"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/utils"
	"mime"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type Tag struct {
	Name string `json:"name"`
	ID   int64  `json:"id"`
}

type GetImageResponse struct {
	sqlc.GetImageRow
	Tags []Tag `json:"tags"`
}

func GetImage(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid id")
		return
	}

	ctx := context.Background()
	img, err := db.Query().GetImage(ctx, id)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when getting image")
		return
	}

	tags, err := db.Query().GetTagsByImage(ctx, img.ID)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when getting tags")
		return
	}

	var tag []Tag = []Tag{}

	for _, it := range tags {
		tag = append(tag, Tag{
			ID:   it.ID,
			Name: it.Name,
		})
	}

	utils.SuccessResponse(c, GetImageResponse{
		GetImageRow: img,
		Tags:        tag,
	})
}

func GetImageVersions(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid id")
		return
	}

	ctx := context.Background()

	ver, err := db.Query().GetImageVersions(ctx, id)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting versions: %v", err)
		return
	}

	utils.SuccessResponse(c, ver)
}

type ListImagesResponse struct {
	Total  int64                `json:"total"`
	Images []sqlc.ListImagesRow `json:"images"`
}

func ListImages(c *gin.Context) {
	offset, limit := utils.Pagination(c)

	ctx := context.Background()

	img, err := db.Query().ListImages(ctx, sqlc.ListImagesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting images: %v", err)
		return
	}

	total, err := db.Query().CountImages(ctx)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when counting images: %v", err)
		return
	}

	utils.SuccessResponse(c, ListImagesResponse{
		Total:  total,
		Images: img,
	})
}

type SearchPayload struct {
	Text        string   `json:"text"`
	IncludeTags []string `json:"includeTags"`
	ExcludeTags []string `json:"excludeTags"`
}

type SearchResponse struct {
	Total  int64 `json:"total"`
	Images []sqlc.FilterImagesRow
}

func SearchImages(c *gin.Context) {
	offset, limit := utils.Pagination(c)

	var payload SearchPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid payload")
		return
	}

	ctx := context.Background()

	img, err := db.Query().FilterImages(ctx, sqlc.FilterImagesParams{
		Limit:       int32(limit),
		Offset:      int32(offset),
		IncludeTags: payload.IncludeTags,
		ExcludeTags: payload.ExcludeTags,
		Text:        payload.Text,
	})

	if err != nil {
		utils.ErrorResponse(c, 400, "Failed when collecting images")
		return
	}

	utils.SuccessResponse(c, SearchResponse{
		Images: img,
	})
}

type QuickSearchPayload struct {
	Keywords []string `json:"keywords"`
}

type QuickSearchResponse struct {
	ID        int64       `json:"id"`
	ImageUrl  string      `json:"imageUrl"`
	Text      string      `json:"text"`
	Rating    sqlc.Rating `json:"rating"`
	Relevance float64     `json:"relevance"`
	Reason    string      `json:"reason"`
}

func QuickSearchImages(c *gin.Context) {
	var payload QuickSearchPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid payload")
		return
	}

	ctx := context.Background()

	var result []QuickSearchResponse

	userID, ok := c.Get("UserID")

	if ok {
		fav, err := db.Query().QuickSearchFavorites(ctx, sqlc.QuickSearchFavoritesParams{
			Keywords: payload.Keywords,
			UserID:   userID.(int64),
		})
		if err != nil {
			utils.ErrorResponse(c, 500, "Failed when collecting favorites")
			return
		}

		for i := range fav {
			result = append(result, QuickSearchResponse{
				ID:        fav[i].ID,
				ImageUrl:  fav[i].ImageUrl,
				Relevance: fav[i].Relevance,
				Reason:    "favorite",
				Text:      fav[i].Text,
				Rating:    fav[i].Rating,
			})
		}
	}

	img, err := db.Query().QuickSearchImage(ctx, payload.Keywords)

	if err != nil {
		utils.ErrorResponse(c, 400, "Failed when collecting image: %v", err)
		return
	}

	for i := range img {
		result = append(result, QuickSearchResponse{
			ID:        img[i].ID,
			ImageUrl:  img[i].ImageUrl,
			Relevance: img[i].Relevance,
			Reason:    "filter",
			Text:      img[i].Text,
			Rating:    img[i].Rating,
		})
	}

	utils.SuccessResponse(c, result)
}

type UpdatePayload struct {
	Text   *string   `json:"text"`
	Rating *string   `json:"rating"`
	Tags   *[]string `json:"tags"`
}

type Metadata struct {
	Text   string
	Rating sqlc.Rating
	Tags   []string
}

type UpdateResponse struct {
	VersionID int64 `json:"versionId"`
}

func UpdateImage(c *gin.Context) {
	permission := c.GetInt64("Permission")

	if permission&constant.EditImage != constant.EditImage {
		utils.ErrorResponse(c, 403, "Operation not allowed")
		return
	}

	imageID, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid image id")
		return
	}

	userID := c.GetInt64("UserID")

	var payload UpdatePayload

	if err = c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid payload")
		return
	}

	ctx := context.Background()

	current, err := db.Query().GetImage(ctx, imageID)

	if err != nil {
		utils.ErrorResponse(c, 500, "Server error: %v", err)
		return
	}

	if current.CurrentVersionID.Valid != true {
		utils.ErrorResponse(c, 500, "Server database corrupted")
		return
	}

	rating := sqlc.Rating(current.Rating)
	text := current.Text
	tags, err := db.Query().GetTagsByVersion(ctx, current.CurrentVersionID.Int64)

	if payload.Text != nil {
		text = *payload.Text
	}
	if payload.Rating != nil {
		rating = sqlc.Rating(*payload.Rating)
	}
	if payload.Tags != nil {
		if len(*payload.Tags) != 0 {
			err := db.Query().CreateTags(ctx, *payload.Tags)
			if err != nil {
				utils.ErrorResponse(c, 500, "Failed when creating tags")
				return
			}
		}

		tags = *payload.Tags
	}

	err = db.Query().CreateNewVersion(ctx, sqlc.CreateNewVersionParams{
		ImageID:  imageID,
		Text:     text,
		Rating:   rating,
		UserID:   userID,
		TagNames: tags,
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when creating new version: %v", err)
		return
	}

	utils.SuccessResponse(c, nil)
}

type SignPayload struct {
	MIME string `json:"mime"`
}

type SignResponse struct {
	SessionID int64  `json:"sessionId"`
	Key       string `json:"key"`
	URL       string `json:"url"`
}

func CreateUploadSession(c *gin.Context) {
	permission := c.GetInt64("Permission")

	if permission&constant.CreateImage != constant.CreateImage {
		utils.ErrorResponse(c, 403, "Operation not allowed")
		return
	}

	var payload SignPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request")
		return
	}

	client := config.GetS3Client()
	presigner := s3.NewPresignClient(client)

	ctx := context.Background()

	key, err := gonanoid.New()

	if err != nil {
		utils.ErrorResponse(c, 500, "Unable to generate nanoid")
		return
	}

	ext, err := mime.ExtensionsByType(payload.MIME)

	if err != nil || len(ext) == 0 {
		utils.ErrorResponse(c, 400, "Invalid MIME type")
		return
	}

	key = key + ext[0]

	req, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(config.GetConfig().S3.BucketName),
		Key:         aws.String(key),
		ContentType: aws.String(payload.MIME),
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when presigning request: %v", err)
		return
	}

	userID, ok := c.Get("UserID")

	if !ok {
		utils.ErrorResponse(c, 500, "Unexpected error")
		return
	}

	sess, err := db.Query().CreateUploadSession(ctx, sqlc.CreateUploadSessionParams{
		UserID: userID.(int64),
		Key:    key,
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when creating session: %v", err)
		return
	}

	utils.SuccessResponse(c, SignResponse{
		SessionID: sess.ID,
		Key:       key,
		URL:       req.URL,
	})
}

type AckPayload struct {
	SessionID int64    `json:"sessionId"`
	Text      string   `json:"text"`
	Rating    string   `json:"rating"`
	Tags      []string `json:"tags"`
}

type AckResponse struct {
	ID int64 `json:"id"`
}

func AcknowledgeSession(c *gin.Context) {
	permission := c.GetInt64("Permission")

	if permission&constant.CreateImage != constant.CreateImage {
		utils.ErrorResponse(c, 403, "Operation not allowed")
		return
	}

	var payload AckPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid data")
		return
	}

	userID, ok := c.Get("UserID")

	if !ok {
		utils.ErrorResponse(c, 500, "Unexpected error")
		return
	}

	ctx := context.Background()

	sess, err := db.Query().CompleteUploadSession(ctx, sqlc.CompleteUploadSessionParams{
		ID:     payload.SessionID,
		UserID: userID.(int64),
	})

	if err != nil {
		utils.ErrorResponse(c, 409, "Server-side session info mismatch")
		return
	}

	err = db.Query().CreateTags(ctx, payload.Tags)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when creating tags")
		return
	}

	imageID, err := db.Query().CreateImage(ctx, sqlc.CreateImageParams{
		ImageKey: sess.Key,
		ImageUrl: config.GetConfig().S3.URLPrefix + sess.Key,
		UserID:   userID.(int64),
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when creating image: %v", err)
		return
	}

	_, err = db.Query().InitializeVersion(ctx, sqlc.InitializeVersionParams{
		ImageID:  imageID,
		Text:     payload.Text,
		TagNames: payload.Tags,
		Rating:   sqlc.Rating(payload.Rating),
		UserID:   userID.(int64),
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when creating new version: %v", err)
		return
	}

	utils.SuccessResponse(c, AckResponse{
		ID: imageID,
	})
}
