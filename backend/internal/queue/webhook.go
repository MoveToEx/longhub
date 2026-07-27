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
	"fmt"
	"long/internal/config"
	"long/internal/db"
	"long/internal/sqlc"
	"net/http"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	WebhookBatchSize   = 100
	WebhookMaxFailures = 3
)

type DispatchArgs struct {
	ID        int64 `json:"id"`
	Offset    int32 `json:"offset"`
	EventType int64 `json:"eventType"`
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
	WebhookID int64  `json:"webhookId"`
	URL       string `json:"url"`
	Body      string `json:"body"`
}

func NewDispatchTask(id int64, offset int32, event int64) (*asynq.Task, error) {
	payload, err := json.Marshal(DispatchArgs{
		ID:        id,
		Offset:    offset,
		EventType: event,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeDispatch, payload), nil
}

func EnqueueDispatch(ctx context.Context, imageID int64, event int64) error {
	task, err := NewDispatchTask(imageID, 0, event)
	if err != nil {
		return err
	}

	_, err = client.EnqueueContext(ctx, task, asynq.Unique(time.Minute))
	if errors.Is(err, asynq.ErrDuplicateTask) {
		return nil
	}
	return err
}

func EnqueueInvoke(ctx context.Context, webhookID int64, body RequestBody) error {
	task, err := NewInvokeTask(webhookID, body)

	if err != nil {
		return err
	}

	_, err = client.EnqueueContext(ctx, task)

	if err != nil {
		return err
	}

	return nil
}

func NewInvokeTask(id int64, body RequestBody) (*asynq.Task, error) {
	s, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(InvokeArgs{
		WebhookID: id,
		Body:      string(s),
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

	count, err := db.Query().CountAvailableWebhooks(ctx, sqlc.CountAvailableWebhooksParams{
		FailureCount: WebhookMaxFailures,
		EventType:    args.EventType,
	})

	if err != nil {
		return err
	}

	webhooks, err := db.Query().GetWebhooksByEvent(ctx, sqlc.GetWebhooksByEventParams{
		Offset:       args.Offset,
		Limit:        WebhookBatchSize,
		FailureCount: WebhookMaxFailures,
		EventType:    args.EventType,
	})

	if err != nil {
		return err
	}

	image, err := db.Query().GetImage(ctx, args.ID)

	if err != nil {
		return err
	}

	tags, err := db.Query().GetTagsByImage(ctx, image.ID)

	if err != nil {
		return err
	}

	names := []string{}

	for i := range tags {
		names = append(names, tags[i].Name)
	}

	body := RequestBody{
		ImageID:   image.ID,
		Text:      image.Text,
		Rating:    image.Rating,
		ImageURL:  image.ImageUrl,
		Tags:      names,
		CreatedAt: image.CreatedAt.Time,
	}

	for i := range webhooks {
		err := EnqueueInvoke(ctx, webhooks[i].ID, body)
		if err != nil {
			return err
		}
	}

	if args.Offset+WebhookBatchSize < int32(count) {
		task, err := NewDispatchTask(args.ID, args.Offset, args.EventType)
		if err != nil {
			return err
		}

		_, err = client.EnqueueContext(ctx, task)

		if err != nil {
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

	webhook, err := db.Query().GetWebhook(ctx, args.WebhookID)

	if err != nil {
		return err
	}

	hash := hmac.New(sha256.New, []byte(webhook.Secret))

	_, err = hash.Write([]byte(args.Body))
	if err != nil {
		return err
	}

	signature := hex.EncodeToString(hash.Sum(nil))

	body := WorkerRequest{
		URL:             webhook.Endpoint,
		Body:            args.Body,
		ClientSignature: signature,
	}

	b, err := json.Marshal(body)

	if err != nil {
		return err
	}

	serverSig := ed25519.Sign(config.GetConfig().Webhook.PrivateKey, b)

	fmt.Printf("sending request to %s\n", config.GetConfig().Webhook.Endpoint)

	req, err := http.NewRequestWithContext(ctx, "POST", config.GetConfig().Webhook.Endpoint, bytes.NewReader(b))

	if err != nil {
		return err
	}

	req.Header.Add("X-Server-Signature", base64.StdEncoding.EncodeToString(serverSig))

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	status := res.StatusCode

	if !(status >= 200 && status < 300) {
		db.Query().RecordWebhookFailure(ctx, sqlc.RecordWebhookFailureParams{
			ID: args.WebhookID,
			LastResponseStatus: pgtype.Int4{
				Valid: true,
				Int32: int32(status),
			},
		})
	} else {
		db.Query().RecordWebhookSuccess(ctx, sqlc.RecordWebhookSuccessParams{
			ID: args.WebhookID,
			LastResponseStatus: pgtype.Int4{
				Valid: true,
				Int32: int32(status),
			},
		})
	}
	return nil
}
