package handlers

import (
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"long/internal/config"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

const syncBatchSize = 1000

type catalogSnapshot func(context.Context, func(GetImageResponse) error) error
type snapshotTransaction func(context.Context, func(*sqlc.Queries) error) error

type syncManifest struct {
	Type          string    `json:"type"`
	SchemaVersion int       `json:"schemaVersion"`
	ExportID      string    `json:"exportId"`
	StartedAt     time.Time `json:"startedAt"`
}

type syncImage struct {
	Type string           `json:"type"`
	Data GetImageResponse `json:"data"`
}

type syncEnd struct {
	Type        string `json:"type"`
	ExportID    string `json:"exportId"`
	RecordCount int64  `json:"recordCount"`
	SHA256      string `json:"sha256"`
}

func streamCatalog(ctx context.Context, transact snapshotTransaction, emit func(GetImageResponse) error) error {
	return transact(ctx, func(q *sqlc.Queries) error {
		var afterID int64
		for {
			rows, err := q.ListSyncImages(ctx, sqlc.ListSyncImagesParams{AfterID: afterID, PageLimit: syncBatchSize})
			if err != nil {
				return err
			}
			for _, row := range rows {
				tags := make([]Tag, 0, len(row.TagIds))
				for i, id := range row.TagIds {
					tags = append(tags, Tag{ID: id, Name: row.TagNames[i]})
				}
				if err := emit(GetImageResponse{
					GetImageRow: sqlc.GetImageRow{
						ID: row.ID, ImageUrl: row.ImageUrl, ImageKey: row.ImageKey, CreatedAt: row.CreatedAt,
						Text: row.Text, Rating: row.Rating, Version: row.Version, UserIdentifier: row.UserIdentifier,
						CurrentVersionID: pgtype.Int8{Int64: row.CurrentVersionID, Valid: true},
					},
					Tags: tags,
				}); err != nil {
					return err
				}
				afterID = row.ID
			}
			if len(rows) < syncBatchSize {
				return nil
			}
		}
	})
}

func NewFullSyncHandler(options config.SyncConfig) gin.HandlerFunc {
	return newFullSyncHandler(func(ctx context.Context, emit func(GetImageResponse) error) error {
		return streamCatalog(ctx, db.ReadSnapshot, emit)
	}, options)
}

func newFullSyncHandler(snapshot catalogSnapshot, options config.SyncConfig) gin.HandlerFunc {
	if options.MaxConcurrent <= 0 {
		options.MaxConcurrent = 2
	}
	if options.ExportTimeout <= 0 {
		options.ExportTimeout = 5 * time.Minute
	}
	slots := make(chan struct{}, options.MaxConcurrent)
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			c.Header("Retry-After", "5")
			utils.ErrorResponse(c, http.StatusTooManyRequests, "Too many active exports; retry later")
			return
		}

		startedAt := time.Now().UTC()
		ctx, cancel := context.WithTimeout(c.Request.Context(), options.ExportTimeout)
		defer cancel()
		controller := http.NewResponseController(c.Writer)
		// A context alone cannot interrupt a write blocked on a slow client.
		_ = controller.SetWriteDeadline(startedAt.Add(options.ExportTimeout))
		defer controller.SetWriteDeadline(time.Time{})

		exportID := rand.Text()
		digest := sha256.New()
		var count int64
		var compressed *gzip.Writer
		var output io.Writer = c.Writer
		started := false
		write := func(value any) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return json.NewEncoder(output).Encode(value)
		}
		start := func() error {
			if started {
				return nil
			}
			started = true
			c.Header("Content-Type", "application/x-ndjson")
			c.Header("X-Accel-Buffering", "no")
			c.Writer.Header().Add("Vary", "Accept-Encoding")
			if acceptsGzip(c.GetHeader("Accept-Encoding")) {
				c.Header("Content-Encoding", "gzip")
				compressed = gzip.NewWriter(c.Writer)
				output = compressed
			}
			return write(syncManifest{Type: "manifest", SchemaVersion: 1, ExportID: exportID, StartedAt: startedAt})
		}
		err := snapshot(ctx, func(image GetImageResponse) error {
			if err := start(); err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			line, err := json.Marshal(syncImage{Type: "image", Data: image})
			if err != nil {
				return err
			}
			line = append(line, '\n')
			if _, err := io.MultiWriter(output, digest).Write(line); err != nil {
				return err
			}
			count++
			if count%100 == 0 {
				if compressed != nil {
					if err := compressed.Flush(); err != nil {
						return err
					}
				}
				return controller.Flush()
			}
			return nil
		})
		if err == nil {
			err = start()
		}
		if err == nil {
			err = write(syncEnd{Type: "end", ExportID: exportID, RecordCount: count, SHA256: hex.EncodeToString(digest.Sum(nil))})
		}
		if compressed != nil {
			if closeErr := compressed.Close(); err == nil {
				err = closeErr
			}
		}
		if err != nil {
			log.Printf("catalog export %s aborted after %d records in %s: %v", exportID, count, time.Since(startedAt), err)
			if !started {
				utils.ErrorResponse(c, http.StatusInternalServerError, "Unable to export catalog")
			}
			return
		}
		log.Printf("catalog export %s completed: %d records in %s", exportID, count, time.Since(startedAt))
	}
}

func acceptsGzip(header string) bool {
	wildcard := false
	for _, encoding := range strings.Split(header, ",") {
		parts := strings.Split(encoding, ";")
		quality := 1.0
		for _, parameter := range parts[1:] {
			key, value, ok := strings.Cut(strings.TrimSpace(parameter), "=")
			if ok && strings.EqualFold(key, "q") {
				var err error
				quality, err = strconv.ParseFloat(value, 64)
				if err != nil || quality < 0 || quality > 1 {
					quality = 0
				}
			}
		}
		switch strings.ToLower(strings.TrimSpace(parts[0])) {
		case "gzip":
			return quality > 0
		case "*":
			wildcard = quality > 0
		}
	}
	return wildcard
}
