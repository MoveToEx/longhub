package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"log"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/utils"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
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
	reason := strings.TrimSpace(payload.Reason)
	if reason == "" || utf8.RuneCountInString(reason) > 2000 {
		utils.ErrorResponse(c, 400, "reason must contain between 1 and 2000 characters")
		return
	}
	ctx := c.Request.Context()
	userID := c.GetInt64("UserID")
	conflict := errors.New("an active request already exists or the image is deleted")
	var requestID int64
	err = db.Transaction(ctx, func(q *sqlc.Queries) error {
		image, err := q.LockImageForDeletion(ctx, id)
		if err != nil {
			return err
		}
		if image.DeletedAt.Valid {
			return conflict
		}
		exists, err := q.HasActiveDeletionRequest(ctx, sqlc.HasActiveDeletionRequestParams{ImageID: id, UserID: userID})
		if err != nil {
			return err
		}
		if exists {
			return conflict
		}
		requestID, err = q.CreateDeletionRequest(ctx, sqlc.CreateDeletionRequestParams{ImageID: id, UserID: userID, Reason: reason})
		return err
	})
	if err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.Is(err, conflict) || errors.As(err, &pgErr) && pgErr.Code == "23505":
			utils.ErrorResponse(c, 409, "An active request already exists or the image is deleted")
		case errors.Is(err, pgx.ErrNoRows):
			utils.ErrorResponse(c, 404, "Image not found")
		default:
			log.Printf("create deletion request: %v", err)
			utils.ErrorResponse(c, 500, "Unable to create deletion request")
		}
		return
	}
	row, err := db.Query().GetDeletionRequestRecord(ctx, requestID)
	if err != nil {
		utils.ErrorResponse(c, 500, "Unable to load deletion request")
		return
	}
	utils.CreatedResponse(c, publicDeletion(row))
}
