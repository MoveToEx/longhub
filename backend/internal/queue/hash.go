package queue

import (
	"context"
	"encoding/json"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"long/internal/db"
	"net/http"

	"github.com/corona10/goimagehash"
	"github.com/hibiken/asynq"
	_ "golang.org/x/image/webp"
)

type HashArgs struct {
	ID int64 `json:"id"`
}

func NewHashTask(id int64) (*asynq.Task, error) {
	payload, err := json.Marshal(HashArgs{ID: id})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeHash, payload), nil
}

func HandleHashTask(ctx context.Context, task *asynq.Task) error {
	var args HashArgs
	if err := json.Unmarshal(task.Payload(), &args); err != nil {
		return err
	}

	img, err := db.Query().GetImage(ctx, args.ID)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, img.ImageUrl, nil)
	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	decoded, _, err := image.Decode(res.Body)
	if err != nil {
		return err
	}

	_, err = goimagehash.DifferenceHash(decoded)
	if err != nil {
		return err
	}

	return nil
}
