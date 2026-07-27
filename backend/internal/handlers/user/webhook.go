package user

import (
	"errors"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type ListWebhooksResponse struct {
	Label              string    `json:"label"`
	EventTypes         int64     `json:"eventTypes"`
	Endpoint           string    `json:"endpoint"`
	Active             bool      `json:"active"`
	LastActivatedAt    time.Time `json:"lastActivatedAt"`
	LastResponseStatus *int32    `json:"lastResponseStatus"`
}

func ListWebhooks(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("UserID")

	webhooks, err := db.Query().GetWebhooksByUser(ctx, userID)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when getting webhooks")
		return
	}

	result := []ListWebhooksResponse{}

	for i := range webhooks {
		var status *int32 = nil

		if webhooks[i].LastResponseStatus.Valid {
			status = new(webhooks[i].LastResponseStatus.Int32)
		}
		result = append(result, ListWebhooksResponse{
			Label:              webhooks[i].Label,
			EventTypes:         webhooks[i].EventTypes,
			Endpoint:           webhooks[i].Endpoint,
			Active:             webhooks[i].Active,
			LastActivatedAt:    webhooks[i].LastActivatedAt.Time,
			LastResponseStatus: status,
		})
	}

	utils.SuccessResponse(c, result)
}

type CreateWebhookPayload struct {
	Label      string `json:"label"`
	EventTypes int64  `json:"eventTypes"`
	Secret     string `json:"secret"`
	Endpoint   string `json:"endpoint"`
}

type CreateWebhookResponse struct {
	ID int64 `json:"id"`
}

func CreateWebhook(c *gin.Context) {
	var payload CreateWebhookPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Failed when parsing body")
		return
	}

	ctx := c.Request.Context()
	userID := c.GetInt64("UserID")

	cnt, err := db.Query().CountWebhooksByUser(ctx, userID)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when counting webhooks")
		return
	}

	if cnt >= 10 {
		utils.ErrorResponse(c, 400, "Too many webhooks created")
		return
	}

	hook, err := db.Query().NewWebhook(ctx, sqlc.NewWebhookParams{
		UserID:     userID,
		Label:      payload.Label,
		Secret:     payload.Secret,
		EventTypes: payload.EventTypes,
		Endpoint:   payload.Endpoint,
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when creating webhook")
		return
	}

	utils.CreatedResponse(c, CreateAppKeyResponse{
		ID: hook.ID,
	})
}

type EditWebhookPayload struct {
	ID         int64   `uri:"id"`
	Label      *string `json:"label"`
	EventTypes *int64  `json:"eventTypes"`
	Secret     *string `json:"secret"`
	Endpoint   *string `json:"endpoint"`
}

func EditWebhook(c *gin.Context) {
	var payload EditWebhookPayload
	if err := c.ShouldBindUri(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request")
		return
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request")
		return
	}

	userID := c.GetInt64("UserID")
	ctx := c.Request.Context()

	webhook, err := db.Query().GetWebhook(ctx, payload.ID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.ErrorResponse(c, 404, "Webhook does not exist")
		} else {
			utils.ErrorResponse(c, 500, "Failed when getting webhook")
		}
		return
	}

	if webhook.UserID != userID {
		utils.ErrorResponse(c, 403, "Ownership mismatch")
		return
	}

	args := sqlc.UpdateWebhookParams{
		Label:      webhook.Label,
		Endpoint:   webhook.Endpoint,
		EventTypes: webhook.EventTypes,
		Secret:     webhook.Secret,
		ID:         webhook.ID,
	}

	if payload.Endpoint != nil {
		args.Endpoint = *payload.Endpoint
	}
	if payload.EventTypes != nil {
		args.EventTypes = *payload.EventTypes
	}
	if payload.Label != nil {
		args.Label = *payload.Label
	}
	if payload.Secret != nil {
		args.Secret = *payload.Secret
	}

	err = db.Query().UpdateWebhook(ctx, args)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when updating webhook")
		return
	}

	utils.SuccessResponse(c, nil)
}

type DeleteWebhookPayload struct {
	ID int64 `uri:"id"`
}

func DeleteWebhook(c *gin.Context) {
	var payload DeleteWebhookPayload

	if err := c.ShouldBindUri(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request")
		return
	}

	userID := c.GetInt64("UserID")
	ctx := c.Request.Context()

	hook, err := db.Query().GetWebhook(ctx, payload.ID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.ErrorResponse(c, 404, "Webhook not found")
		} else {
			utils.ErrorResponse(c, 500, "Failed when getting webhook")
		}
		return
	}

	if hook.UserID != userID {
		utils.ErrorResponse(c, 403, "Webhook ownership mismatch")
		return
	}

	err = db.Query().DeleteWebhook(ctx, payload.ID)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when deleting webhook")
		return
	}

	utils.SuccessResponse(c, nil)
}
