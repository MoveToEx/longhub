package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/utils"
	webhookpayload "long/internal/webhook"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ListWebhooksResponse struct {
	ID                 int64             `json:"id"`
	Label              string            `json:"label"`
	EventTypes         int64             `json:"eventTypes"`
	Endpoint           string            `json:"endpoint"`
	Active             bool              `json:"active"`
	LastActivatedAt    *time.Time        `json:"lastActivatedAt"`
	LastResponseStatus *int32            `json:"lastResponseStatus"`
	FailureCount       int32             `json:"failureCount"`
	BodyTemplate       *string           `json:"bodyTemplate"`
	HeaderTemplates    map[string]string `json:"headerTemplates"`
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
		var activatedAt *time.Time = nil
		var bodyTemplate *string

		if webhooks[i].LastResponseStatus.Valid {
			status = new(webhooks[i].LastResponseStatus.Int32)
		}
		if webhooks[i].LastActivatedAt.Valid {
			activatedAt = new(webhooks[i].LastActivatedAt.Time)
		}
		if webhooks[i].BodyTemplate.Valid {
			bodyTemplate = &webhooks[i].BodyTemplate.String
		}
		headers := webhooks[i].HeaderTemplates
		if headers == nil {
			headers = map[string]string{}
		}
		result = append(result, ListWebhooksResponse{
			ID:                 webhooks[i].ID,
			Label:              webhooks[i].Label,
			EventTypes:         webhooks[i].EventTypes,
			Endpoint:           webhooks[i].Endpoint,
			Active:             webhooks[i].Active,
			LastActivatedAt:    activatedAt,
			LastResponseStatus: status,
			FailureCount:       webhooks[i].FailureCount,
			BodyTemplate:       bodyTemplate,
			HeaderTemplates:    headers,
		})
	}

	utils.SuccessResponse(c, result)
}

type CreateWebhookPayload struct {
	Label           string          `json:"label"`
	EventTypes      int64           `json:"eventTypes"`
	Secret          string          `json:"secret"`
	Endpoint        string          `json:"endpoint"`
	Active          bool            `json:"active"`
	BodyTemplate    *string         `json:"bodyTemplate"`
	HeaderTemplates json.RawMessage `json:"headerTemplates"`
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
	bodyTemplate := nullableBodyTemplate(payload.BodyTemplate)
	if err := validateBodyTemplate(bodyTemplate, payload.EventTypes); err != nil {
		utils.ErrorResponse(c, 400, "%s", err.Error())
		return
	}
	headers, err := decodeHeaderTemplates(payload.HeaderTemplates)
	if err != nil {
		utils.ErrorResponse(c, 400, "%s", err.Error())
		return
	}
	if err := validateHeaderTemplates(headers, payload.EventTypes); err != nil {
		utils.ErrorResponse(c, 400, "%s", err.Error())
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
		UserID:          userID,
		Label:           payload.Label,
		Secret:          payload.Secret,
		EventTypes:      payload.EventTypes,
		Endpoint:        payload.Endpoint,
		Active:          payload.Active,
		BodyTemplate:    bodyTemplate,
		HeaderTemplates: encodeHeaderTemplates(headers),
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when creating webhook")
		return
	}

	utils.CreatedResponse(c, CreateWebhookResponse{
		ID: hook.ID,
	})
}

type EditWebhookPayload struct {
	ID              int64           `uri:"id"`
	Label           *string         `json:"label"`
	EventTypes      *int64          `json:"eventTypes"`
	Secret          *string         `json:"secret"`
	Endpoint        *string         `json:"endpoint"`
	Active          *bool           `json:"active"`
	BodyTemplate    json.RawMessage `json:"bodyTemplate"`
	HeaderTemplates json.RawMessage `json:"headerTemplates"`
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
		Label:           webhook.Label,
		Endpoint:        webhook.Endpoint,
		EventTypes:      webhook.EventTypes,
		Secret:          webhook.Secret,
		ID:              webhook.ID,
		Active:          webhook.Active,
		BodyTemplate:    webhook.BodyTemplate,
		HeaderTemplates: encodeHeaderTemplates(webhook.HeaderTemplates),
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
	if payload.Active != nil {
		args.Active = *payload.Active
	}
	if payload.BodyTemplate != nil {
		var value *string
		if err := json.Unmarshal(payload.BodyTemplate, &value); err != nil {
			utils.ErrorResponse(c, 400, "Body template must be a string or null")
			return
		}
		args.BodyTemplate = nullableBodyTemplate(value)
	}
	if payload.BodyTemplate != nil || payload.EventTypes != nil {
		if err := validateBodyTemplate(args.BodyTemplate, args.EventTypes); err != nil {
			utils.ErrorResponse(c, 400, "%s", err.Error())
			return
		}
	}
	headers := webhook.HeaderTemplates
	if payload.HeaderTemplates != nil {
		headers, err = decodeHeaderTemplates(payload.HeaderTemplates)
		if err != nil {
			utils.ErrorResponse(c, 400, "%s", err.Error())
			return
		}
		args.HeaderTemplates = encodeHeaderTemplates(headers)
	}
	if payload.HeaderTemplates != nil || payload.EventTypes != nil {
		if err := validateHeaderTemplates(headers, args.EventTypes); err != nil {
			utils.ErrorResponse(c, 400, "%s", err.Error())
			return
		}
	}

	err = db.Query().UpdateWebhook(ctx, args)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when updating webhook")
		return
	}

	utils.SuccessResponse(c, nil)
}

func nullableBodyTemplate(value *string) pgtype.Text {
	if value == nil || strings.TrimSpace(*value) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func validateBodyTemplate(value pgtype.Text, eventTypes int64) error {
	if !value.Valid {
		return nil
	}
	return validateWebhookEvents(eventTypes, func(event string) error {
		_, err := webhookpayload.Render(value.String, event, webhookpayload.PreviewData(event), webhookpayload.PreviewInitiator(event))
		if err != nil {
			return fmt.Errorf("Invalid body template for %s: %w", event, err)
		}
		return nil
	})
}

func decodeHeaderTemplates(raw json.RawMessage) (map[string]string, error) {
	if raw == nil {
		return nil, nil
	}
	var values map[string]*string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, errors.New("Header templates must be an object with string values or null")
	}
	headers := make(map[string]string, len(values))
	for name, value := range values {
		if value == nil {
			return nil, fmt.Errorf("Template for header %q must be a string", name)
		}
		headers[name] = *value
	}
	return headers, nil
}

func encodeHeaderTemplates(headers map[string]string) []byte {
	if headers == nil {
		return []byte("{}")
	}
	// A map of strings always encodes successfully.
	encoded, _ := json.Marshal(headers)
	return encoded
}

func validateHeaderTemplates(headers map[string]string, eventTypes int64) error {
	return validateWebhookEvents(eventTypes, func(event string) error {
		_, err := webhookpayload.RenderHeaders(headers, event, webhookpayload.PreviewData(event), webhookpayload.PreviewInitiator(event))
		if err != nil {
			return fmt.Errorf("Invalid header templates for %s: %w", event, err)
		}
		return nil
	})
}

func validateWebhookEvents(eventTypes int64, validate func(string) error) error {
	eventTypes &= db.WebhookCreationEvent | db.WebhookUpdateEvent | db.WebhookDeletionEvent
	for _, event := range []struct {
		flag int64
		name string
	}{
		{db.WebhookCreationEvent, "create"},
		{db.WebhookUpdateEvent, "update"},
		{db.WebhookDeletionEvent, "delete"},
		{db.WebhookDeletionEvent, "restore"},
	} {
		if eventTypes&event.flag == 0 && eventTypes != 0 {
			continue
		}
		if err := validate(event.name); err != nil {
			return err
		}
	}
	return nil
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
