package queue

import (
	"context"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"long/internal/db"
	"net/http"

	"github.com/corona10/goimagehash"
	"github.com/riverqueue/river"
	_ "golang.org/x/image/webp"
)

type HashArgs struct {
	ID int64 `json:"id"`
}

func (HashArgs) Kind() string {
	return "hash"
}

type HashWorker struct {
	river.WorkerDefaults[HashArgs]
}

func (w *HashWorker) Work(ctx context.Context, job *river.Job[HashArgs]) error {
	img, err := db.Query().GetImage(ctx, job.Args.ID)
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
