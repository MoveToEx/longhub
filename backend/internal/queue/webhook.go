package queue

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"long/internal/config"
	"long/internal/db"
	"long/internal/sqlc"
	webhookpayload "long/internal/webhook"
	"net/http"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	WebhookBatchSize      = 100
	WebhookMaxFailures    = 3
	WebhookRequestTimeout = 30 * time.Second
)

var webhookHTTPClient = &http.Client{
	Timeout: WebhookRequestTimeout,
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

type DispatchArgs struct {
	ID            int64 `json:"id"`
	VersionID     int64 `json:"version"`
	LastWebhookID int64 `json:"lastWebhookId"`
	EventType     int64 `json:"eventType"`
}

type RequestBody = webhookpayload.Data

type WorkerRequest struct {
	URL             string `json:"url"`
	Body            string `json:"body"`
	ClientSignature string `json:"clientSignature"`
}

type InvocationBody struct {
	Event     string               `json:"event"`
	Data      json.RawMessage      `json:"data"`
	Initiator *webhookpayload.User `json:"initiator,omitempty"`
}

type InvokeArgs struct {
	WebhookID int64                `json:"webhookId"`
	ImageID   int64                `json:"imageId"`
	Version   int32                `json:"version"`
	EventType int64                `json:"eventType"`
	Body      json.RawMessage      `json:"body"`
	Initiator *webhookpayload.User `json:"initiator,omitempty"`
}

func shouldIgnoreInvocation(currentVersion, dispatchedVersion int32) bool {
	return currentVersion > dispatchedVersion
}

func decodeInvokeBody(raw json.RawMessage) ([]byte, error) {
	if len(raw) > 0 && raw[0] == '"' {
		var body string
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, err
		}
		return []byte(body), nil
	}
	return raw, nil
}

func webhookEventName(eventType int64) (string, error) {
	switch eventType {
	case db.WebhookCreationEvent:
		return "create", nil
	case db.WebhookUpdateEvent:
		return "update", nil
	case db.WebhookDeletionEvent:
		return "delete", nil
	default:
		return "", errors.New("invalid webhook event type")
	}
}

func buildInvocationBody(eventType int64, data []byte, bodyTemplate string, initiator *webhookpayload.User) ([]byte, error) {
	event, err := webhookEventName(eventType)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(bodyTemplate) != "" {
		var body RequestBody
		if err := json.Unmarshal(data, &body); err != nil {
			return nil, err
		}
		return webhookpayload.Render(bodyTemplate, event, body, initiator)
	}

	return json.Marshal(InvocationBody{
		Event:     event,
		Data:      json.RawMessage(data),
		Initiator: initiator,
	})
}

func NewDispatchTask(id, versionID, lastWebhookID int64, event int64) (*asynq.Task, error) {
	payload, err := json.Marshal(DispatchArgs{
		ID:            id,
		VersionID:     versionID,
		LastWebhookID: lastWebhookID,
		EventType:     event,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeDispatch, payload), nil
}

func EnqueueDispatch(ctx context.Context, imageID, versionID int64, event int64) error {
	task, err := NewDispatchTask(imageID, versionID, 0, event)
	if err != nil {
		return err
	}

	_, err = client.EnqueueContext(ctx, task, asynq.Unique(time.Minute))
	if errors.Is(err, asynq.ErrDuplicateTask) {
		return nil
	}
	return err
}

func EnqueueInvoke(ctx context.Context, webhookID, imageID int64, version int32, eventType int64, body RequestBody, initiator *webhookpayload.User) error {
	task, err := NewInvokeTask(webhookID, imageID, version, eventType, body, initiator)

	if err != nil {
		return err
	}

	_, err = client.EnqueueContext(ctx, task, asynq.Unique(time.Hour))

	if err != nil {
		if errors.Is(err, asynq.ErrDuplicateTask) {
			return nil
		}
		return err
	}
	return nil
}

func NewInvokeTask(id, imageID int64, version int32, eventType int64, body RequestBody, initiator *webhookpayload.User) (*asynq.Task, error) {
	payloadBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(InvokeArgs{
		WebhookID: id,
		ImageID:   imageID,
		Version:   version,
		EventType: eventType,
		Body:      payloadBody,
		Initiator: initiator,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeInvoke, payload), nil
}

func HandleDispatchTask(ctx context.Context, task *asynq.Task) error {
	var args DispatchArgs
	if err := json.Unmarshal(task.Payload(), &args); err != nil {
		return err
	}

	webhooks, err := db.Query().GetWebhooksByEvent(ctx, sqlc.GetWebhooksByEventParams{
		AfterID:      args.LastWebhookID,
		PageLimit:    WebhookBatchSize,
		FailureCount: WebhookMaxFailures,
		EventType:    args.EventType,
	})

	if err != nil {
		return err
	}

	image, err := db.Query().GetImageVersionForWebhook(ctx, sqlc.GetImageVersionForWebhookParams{
		ImageID:   args.ID,
		VersionID: args.VersionID,
	})

	if err != nil {
		return err
	}

	body := RequestBody{
		ImageID:   image.ImageID,
		Text:      image.Text,
		Rating:    image.Rating,
		ImageURL:  image.ImageUrl,
		Tags:      image.Tags,
		CreatedAt: image.CreatedAt.Time,
		Uploader: &webhookpayload.User{
			ID: image.UploaderID, Username: image.UploaderUsername,
		},
	}
	initiator := &webhookpayload.User{
		ID: image.InitiatorID, Username: image.InitiatorUsername,
	}
	if args.EventType == db.WebhookUpdateEvent {
		previous, err := db.Query().GetPreviousImageVersionForWebhook(ctx, sqlc.GetPreviousImageVersionForWebhookParams{
			ImageID: image.ImageID,
			Version: image.Version,
		})
		if err != nil {
			return err
		}
		body.Changes = webhookpayload.Compare(RequestBody{
			Text: previous.Text, Rating: previous.Rating, Tags: previous.Tags,
		}, body)
	}

	for i := range webhooks {
		err := EnqueueInvoke(ctx, webhooks[i].ID, image.ImageID, image.Version, args.EventType, body, initiator)
		if err != nil {
			return err
		}
	}

	if len(webhooks) == WebhookBatchSize {
		task, err := NewDispatchTask(args.ID, args.VersionID, webhooks[len(webhooks)-1].ID, args.EventType)
		if err != nil {
			return err
		}

		_, err = client.EnqueueContext(ctx, task, asynq.Unique(time.Hour))

		if err != nil {
			if errors.Is(err, asynq.ErrDuplicateTask) {
				return nil
			}
			return err
		}
	}

	return nil
}

func HandleInvokeTask(ctx context.Context, task *asynq.Task) error {
	var args InvokeArgs
	if err := json.Unmarshal(task.Payload(), &args); err != nil {
		return err
	}

	image, err := db.Query().GetImage(ctx, args.ImageID)
	if err != nil {
		return err
	}
	if shouldIgnoreInvocation(image.Version, args.Version) {
		return nil
	}
	bodyBytes, err := decodeInvokeBody(args.Body)
	if err != nil {
		return err
	}
	webhook, err := db.Query().GetWebhook(ctx, args.WebhookID)

	if err != nil {
		return err
	}
	if !webhook.Active {
		return nil
	}
	invocationBody, err := buildInvocationBody(args.EventType, bodyBytes, webhook.BodyTemplate.String, args.Initiator)
	if err != nil {
		recordErr := db.Query().RecordWebhookFailure(ctx, sqlc.RecordWebhookFailureParams{
			ID:           args.WebhookID,
			FailureCount: WebhookMaxFailures,
		})
		return errors.Join(err, recordErr)
	}

	hash := hmac.New(sha256.New, []byte(webhook.Secret))

	_, err = hash.Write(invocationBody)
	if err != nil {
		return err
	}

	signature := hex.EncodeToString(hash.Sum(nil))

	body := WorkerRequest{
		URL:             webhook.Endpoint,
		Body:            string(invocationBody),
		ClientSignature: signature,
	}

	b, err := json.Marshal(body)

	if err != nil {
		return err
	}

	serverSig := ed25519.Sign(config.GetConfig().Webhook.PrivateKey, b)

	log.Printf("sending webhook request to %s", config.GetConfig().Webhook.Endpoint)

	req, err := http.NewRequestWithContext(ctx, "POST", config.GetConfig().Webhook.Endpoint, bytes.NewReader(b))

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Server-Signature", base64.StdEncoding.EncodeToString(serverSig))

	res, err := webhookHTTPClient.Do(req)

	if err != nil {
		recordErr := db.Query().RecordWebhookFailure(ctx, sqlc.RecordWebhookFailureParams{
			ID:           args.WebhookID,
			FailureCount: WebhookMaxFailures,
		})
		if recordErr != nil {
			return errors.Join(err, recordErr)
		}
		return err
	}

	defer res.Body.Close()

	status := res.StatusCode

	if status < 200 || status >= 300 {
		return db.Query().RecordWebhookFailure(ctx, sqlc.RecordWebhookFailureParams{
			ID: args.WebhookID,
			LastResponseStatus: pgtype.Int4{
				Valid: true,
				Int32: int32(status),
			},
			FailureCount: WebhookMaxFailures,
		})
	}
	return db.Query().RecordWebhookSuccess(ctx, sqlc.RecordWebhookSuccessParams{
		ID: args.WebhookID,
		LastResponseStatus: pgtype.Int4{
			Valid: true,
			Int32: int32(status),
		},
	})
}
