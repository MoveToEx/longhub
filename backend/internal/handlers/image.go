package handlers

import (
	"errors"
	"fmt"
	"log"
	"long/internal/config"
	"long/internal/db"
	"long/internal/queue"
	"long/internal/sqlc"
	"long/internal/utils"
	"mime"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/meilisearch/meilisearch-go"
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

	ctx := c.Request.Context()
	img, err := db.Query().GetImage(ctx, id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.ErrorResponse(c, 404, "Image not found")
		} else {
			utils.ErrorResponse(c, 500, "Failed when getting image")
		}
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

	ctx := c.Request.Context()

	ver, err := db.Query().GetImageVersions(ctx, id)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting versions")
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

	ctx := c.Request.Context()

	img, err := db.Query().ListImages(ctx, sqlc.ListImagesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting images")
		return
	}

	total, err := db.Query().CountImages(ctx)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when counting images")
		return
	}

	utils.SuccessResponse(c, ListImagesResponse{
		Total:  total,
		Images: img,
	})
}

type SearchCondition struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type SearchPayload struct {
	Conditions []SearchCondition `json:"conditions"`
	OrderBy    string            `json:"orderBy"`
	Order      string            `json:"order"`
}

type SearchResponse struct {
	Total  int64                    `json:"total"`
	Images []sqlc.GetImagesByIdsRow `json:"images"`
}

func SearchImages(c *gin.Context) {
	offset, limit := utils.Pagination(c)

	var payload SearchPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid payload")
		return
	}

	if len(payload.Conditions) > 50 {
		utils.ErrorResponse(c, 400, "Too many search conditions")
		return
	}

	filters := make([]string, 0, len(payload.Conditions))
	for _, condition := range payload.Conditions {
		value := strings.TrimSpace(condition.Value)
		if value == "" {
			utils.ErrorResponse(c, 400, "Search condition values cannot be empty")
			return
		}

		quoted := quoteMeiliFilterValue(value)
		switch condition.Type {
		case "tagInclude":
			filters = append(filters, "tags = "+quoted)
		case "tagExclude":
			filters = append(filters, "NOT tags = "+quoted)
		case "ratingEq":
			rating, ok := map[string]int{"none": 1, "moderate": 2, "violent": 3}[value]
			if !ok {
				utils.ErrorResponse(c, 400, "Invalid rating")
				return
			}
			filters = append(filters, fmt.Sprintf("rating = %d", rating))
		case "textContains":
			filters = append(filters, "text CONTAINS "+quoted)
		case "uploadedBy":
			if userID, err := strconv.ParseInt(value, 10, 64); err == nil && userID > 0 {
				filters = append(filters, fmt.Sprintf("(uploader = %s OR userId = %d)", quoted, userID))
			} else {
				filters = append(filters, "uploader = "+quoted)
			}
		default:
			utils.ErrorResponse(c, 400, "Invalid search condition")
			return
		}
	}

	orderBy := map[string]string{
		"id":         "id",
		"uploadDate": "createdAt",
	}[payload.OrderBy]
	if orderBy == "" {
		orderBy = "createdAt"
	}
	order := payload.Order
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	if offset < 0 {
		offset = 0
	}
	if limit < 1 {
		limit = 24
	}

	request := &meilisearch.SearchRequest{
		Offset:               int64(offset),
		Limit:                int64(limit),
		AttributesToRetrieve: []string{"id"},
		Sort:                 []string{orderBy + ":" + order},
	}
	if len(filters) > 0 {
		request.Filter = strings.Join(filters, " AND ")
	}

	msr, err := config.MeiliSearch().Index("images").Search("", request)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when searching images")
		return
	}

	ids := make([]int64, 0, len(msr.Hits))
	for i := range msr.Hits {
		var item MeilisearchResult
		if err := msr.Hits[i].DecodeInto(&item); err != nil {
			utils.ErrorResponse(c, 500, "Failed when reading search results")
			return
		}
		ids = append(ids, item.ID)
	}

	images, err := db.Query().GetImagesByIds(c.Request.Context(), ids)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when fetching images")
		return
	}

	utils.SuccessResponse(c, SearchResponse{
		Total:  msr.EstimatedTotalHits,
		Images: images,
	})
}

func quoteMeiliFilterValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

type QuickSearchPayload struct {
	Keywords []string `json:"keywords"`
}

type QuickSearchResponse struct {
	Favorited []QuickSearchImageResponse `json:"favorited"`
	Filtered  []QuickSearchImageResponse `json:"filtered"`
}

type MeilisearchResult struct {
	ID int64 `json:"id"`
}

type QuickSearchImageResponse struct {
	ID       int64       `json:"id"`
	ImageUrl string      `json:"imageUrl"`
	Text     string      `json:"text"`
	Rating   sqlc.Rating `json:"rating"`
}

func QuickSearchImages(c *gin.Context) {
	var payload QuickSearchPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid payload")
		return
	}

	ctx := c.Request.Context()

	result := QuickSearchResponse{
		Favorited: []QuickSearchImageResponse{},
		Filtered:  []QuickSearchImageResponse{},
	}

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
			result.Favorited = append(result.Favorited, QuickSearchImageResponse{
				ID:       fav[i].ID,
				ImageUrl: fav[i].ImageUrl,
				Text:     fav[i].Text,
				Rating:   fav[i].Rating,
			})
		}
	}

	ms := config.MeiliSearch()

	msr, err := ms.Index("images").Search(strings.Join(payload.Keywords, " "), &meilisearch.SearchRequest{
		Limit: 16,
	})
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when searching images")
		return
	}

	ids := []int64{}

	if msr != nil {
		for i := range msr.Hits {
			var item MeilisearchResult

			if err := msr.Hits[i].DecodeInto(&item); err != nil {
				utils.ErrorResponse(c, 500, "Failed when fetching rows")
				return
			}

			ids = append(ids, item.ID)
		}
	}

	im, err := db.Query().GetImagesByIds(ctx, ids)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when fetching rows")
		return
	}

	for i := range im {
		result.Filtered = append(result.Filtered, QuickSearchImageResponse{
			ID:       im[i].ID,
			ImageUrl: im[i].ImageUrl,
			Text:     im[i].Text,
			Rating:   im[i].Rating,
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

	if permission&db.PermissionEdit != db.PermissionEdit {
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

	ctx := c.Request.Context()

	current, err := db.Query().GetImage(ctx, imageID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.ErrorResponse(c, 404, "Image not found")
		} else {
			utils.ErrorResponse(c, 500, "Server error")
		}
		return
	}

	if current.CurrentVersionID.Valid != true {
		utils.ErrorResponse(c, 500, "Server database corrupted")
		return
	}

	rating := sqlc.Rating(current.Rating)
	text := current.Text
	tags, err := db.Query().GetTagsByVersion(ctx, current.CurrentVersionID.Int64)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting current tags")
		return
	}

	if payload.Text != nil {
		text = *payload.Text
	}
	if payload.Rating != nil {
		rating = sqlc.Rating(*payload.Rating)
	}
	if payload.Tags != nil {
		tags = *payload.Tags
	}

	var versionID int64
	err = db.Transaction(ctx, func(tx *sqlc.Queries) error {
		if len(tags) != 0 {
			if err := tx.CreateTags(ctx, tags); err != nil {
				return err
			}
		}

		versionID, err = tx.CreateNewVersion(ctx, sqlc.CreateNewVersionParams{
			ImageID:  imageID,
			Text:     text,
			Rating:   rating,
			UserID:   userID,
			TagNames: tags,
		})
		return err
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when creating new version")
		return
	}

	if err := queue.EnqueueIndex(ctx, imageID, versionID); err != nil {
		fmt.Printf("Failed when scheduling image %d version %d for indexing: %v", imageID, versionID, err)
	}

	if err := queue.EnqueueDispatch(ctx, imageID, versionID, db.WebhookUpdateEvent); err != nil {
		fmt.Printf("Failed when enqueueing webhook dispatch for image %d: %v", imageID, err)
	}

	utils.SuccessResponse(c, nil)
}

type SignPayload struct {
	MIME string `json:"mime" validate:"required"`
	Size int64  `json:"size" validate:"required,gt=0"`
}

type SignResponse struct {
	SessionID int64  `json:"sessionId"`
	Key       string `json:"key"`
	URL       string `json:"url"`
}

func CreateUploadSession(c *gin.Context) {
	permission := c.GetInt64("Permission")

	if permission&db.PermissionCreate != db.PermissionCreate {
		utils.ErrorResponse(c, 403, "Operation not allowed")
		return
	}

	var payload SignPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request")
		return
	}

	if payload.Size > 4*1024*1024 {
		utils.ErrorResponse(c, 400, "Image too large, try compressing")
		return
	}

	client := config.GetS3Client()
	presigner := s3.NewPresignClient(client)

	ctx := c.Request.Context()

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
		Bucket:        aws.String(config.GetConfig().S3.BucketName),
		Key:           aws.String(key),
		ContentType:   aws.String(payload.MIME),
		ContentLength: aws.Int64(payload.Size),
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when presigning request")
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
		utils.ErrorResponse(c, 500, "Failed when creating session")
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

	if permission&db.PermissionCreate != db.PermissionCreate {
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

	ctx := c.Request.Context()

	var id int64
	var versionID int64

	err := db.Transaction(ctx, func(tx *sqlc.Queries) error {
		sess, err := tx.CompleteUploadSession(ctx, sqlc.CompleteUploadSessionParams{
			ID:     payload.SessionID,
			UserID: userID.(int64),
		})

		if err != nil {
			return err
		}
		err = tx.CreateTags(ctx, payload.Tags)

		if err != nil {
			return err
		}

		id, err = tx.CreateImage(ctx, sqlc.CreateImageParams{
			ImageKey: sess.Key,
			ImageUrl: config.GetConfig().S3.URLPrefix + sess.Key,
			UserID:   userID.(int64),
		})

		if err != nil {
			return err
		}

		versionID, err = tx.InitializeVersion(ctx, sqlc.InitializeVersionParams{
			ImageID:  id,
			Text:     payload.Text,
			TagNames: payload.Tags,
			Rating:   sqlc.Rating(payload.Rating),
			UserID:   userID.(int64),
		})

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when creating image")
		return
	}

	if err := queue.EnqueueIndex(ctx, id, versionID); err != nil {
		log.Printf("Failed when scheduling image %d version %d for indexing: %v", id, versionID, err)
	}

	if err := queue.EnqueueDispatch(ctx, id, versionID, db.WebhookCreationEvent); err != nil {
		log.Printf("Failed when enqueueing webhook dispatch for image %d: %v", id, err)
	}

	utils.SuccessResponse(c, AckResponse{
		ID: id,
	})
}
