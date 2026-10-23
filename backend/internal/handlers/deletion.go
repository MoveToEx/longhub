package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"log"
	"long/internal/db"
	"long/internal/deletion"
	"long/internal/sqlc"
	"long/internal/utils"
	"net/http"
	"strconv"
	"time"
)

type PublicDeletionRequest struct {
	ID          int64               `json:"id"`
	ImageID     int64               `json:"imageId"`
	Reason      string              `json:"reason"`
	Status      sqlc.DeletionStatus `json:"status"`
	Requester   User                `json:"requester"`
	CreatedAt   time.Time           `json:"createdAt"`
	ProcessedAt *time.Time          `json:"processedAt"`
}

func publicDeletion(row sqlc.DeletionRequestDetail) PublicDeletionRequest {
	result := PublicDeletionRequest{ID: row.ID, ImageID: row.ImageID, Reason: row.Reason, Status: row.Status,
		Requester: User{ID: row.UserID, Username: row.RequesterUsername}, CreatedAt: row.CreatedAt.Time}
	if row.ProcessedAt.Valid {
		result.ProcessedAt = &row.ProcessedAt.Time
	}
	return result
}

func ListImageDeletionRequests(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		utils.ErrorResponse(c, 400, "Invalid image id")
		return
	}
	offset, limit := 0, 25
	if value := c.Query("offset"); value != "" {
		offset, err = strconv.Atoi(value)
		if err != nil {
			offset = -1
		}
	}
	if value := c.Query("limit"); value != "" {
		limit, err = strconv.Atoi(value)
		if err != nil {
			limit = 0
		}
	}
	if offset < 0 || int64(offset) > 2147483647 || limit < 1 || limit > 100 {
		utils.ErrorResponse(c, 400, "Invalid pagination")
		return
	}
	ctx := c.Request.Context()
	q := db.Query()
	if _, err := q.GetImageDetails(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.ErrorResponse(c, 404, "Image not found")
		} else {
			utils.ErrorResponse(c, 500, "Unable to load deletion requests")
		}
		return
	}
	rows, err := q.ListImageDeletionRequests(ctx, sqlc.ListImageDeletionRequestsParams{ImageID: id, Offset: int32(offset), Limit: int32(limit)})
	if err != nil {
		utils.ErrorResponse(c, 500, "Unable to load deletion requests")
		return
	}
	total, err := q.CountImageDeletionRequests(ctx, id)
	if err != nil {
		utils.ErrorResponse(c, 500, "Unable to load deletion requests")
		return
	}
	items := make([]PublicDeletionRequest, 0, len(rows))
	for _, row := range rows {
		items = append(items, publicDeletion(row))
	}
	utils.SuccessResponse(c, gin.H{"items": items, "total": total})
}

func CreateImageDeletionRequest(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64*1024)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		utils.ErrorResponse(c, 400, "Invalid image id")
		return
	}
	var payload struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request body")
		return
	}
	service := deletion.Service{Transact: db.Transaction}
	requestID, err := service.Create(c.Request.Context(), id, c.GetInt64("UserID"), payload.Reason)
	if err != nil {
		switch {
		case errors.Is(err, deletion.ErrReason):
			utils.ErrorResponse(c, 400, "%s", err)
		case errors.Is(err, deletion.ErrConflict):
			utils.ErrorResponse(c, 409, "An active request already exists or the image is deleted")
		case errors.Is(err, deletion.ErrNotFound):
			utils.ErrorResponse(c, 404, "Image not found")
		default:
			log.Printf("create deletion request: %v", err)
			utils.ErrorResponse(c, 500, "Unable to create deletion request")
		}
		return
	}
	row, err := db.Query().GetDeletionRequestRecord(c.Request.Context(), requestID)
	if err != nil {
		utils.ErrorResponse(c, 500, "Unable to load deletion request")
		return
	}
	utils.CreatedResponse(c, publicDeletion(row))
}
