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
	"net/http"
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

type RequestBody struct {
	ImageID   int64       `json:"imageID"`
	Text      string      `json:"text"`
	Rating    sqlc.Rating `json:"rating"`
	Tags      []string    `json:"tags"`
	ImageURL  string      `json:"imageURL"`
	CreatedAt time.Time   `json:"createdAt"`
}

type WorkerRequest struct {
	URL             string `json:"url"`
	Body            string `json:"body"`
	ClientSignature string `json:"clientSignature"`
}

type InvokeArgs struct {
	WebhookID int64           `json:"webhookId"`
	ImageID   int64           `json:"imageId"`
	Version   int32           `json:"version"`
	Body      json.RawMessage `json:"body"`
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

func EnqueueInvoke(ctx context.Context, webhookID, imageID int64, version int32, body RequestBody) error {
	task, err := NewInvokeTask(webhookID, imageID, version, body)

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

func NewInvokeTask(id, imageID int64, version int32, body RequestBody) (*asynq.Task, error) {
	payloadBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(InvokeArgs{
		WebhookID: id,
		ImageID:   imageID,
		Version:   version,
		Body:      payloadBody,
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
	}

	for i := range webhooks {
		err := EnqueueInvoke(ctx, webhooks[i].ID, image.ImageID, image.Version, body)
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

	hash := hmac.New(sha256.New, []byte(webhook.Secret))

	_, err = hash.Write(bodyBytes)
	if err != nil {
		return err
	}

	signature := hex.EncodeToString(hash.Sum(nil))

	body := WorkerRequest{
		URL:             webhook.Endpoint,
		Body:            string(bodyBytes),
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
