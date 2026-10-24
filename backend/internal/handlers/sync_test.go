package handlers

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"long/internal/config"
	"long/internal/db"
	"long/internal/middleware"
	"long/internal/sqlc"
	"long/internal/testutil"
	"long/internal/utils"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func readSyncExport(t *testing.T, data []byte) []GetImageResponse {
	t.Helper()
	lines := bytes.Split(data, []byte{'\n'})
	if len(lines) < 3 || len(lines[len(lines)-1]) != 0 {
		t.Fatalf("incomplete export: %s", data)
	}
	var manifest syncManifest
	if err := json.Unmarshal(lines[0], &manifest); err != nil || manifest.Type != "manifest" || manifest.SchemaVersion != 1 || manifest.ExportID == "" {
		t.Fatalf("invalid manifest: %+v %v", manifest, err)
	}
	var end syncEnd
	if err := json.Unmarshal(lines[len(lines)-2], &end); err != nil || end.Type != "end" || end.ExportID != manifest.ExportID {
		t.Fatalf("invalid end: %+v %v", end, err)
	}
	images := []GetImageResponse{}
	digest := sha256.New()
	for _, line := range lines[1 : len(lines)-2] {
		var record syncImage
		if err := json.Unmarshal(line, &record); err != nil || record.Type != "image" {
			t.Fatalf("invalid image: %s %v", line, err)
		}
		images = append(images, record.Data)
		digest.Write(line)
		digest.Write([]byte{'\n'})
	}
	if end.RecordCount != int64(len(images)) || end.SHA256 != hex.EncodeToString(digest.Sum(nil)) {
		t.Fatalf("export verification failed: %+v", end)
	}
	return images
}

func TestFullSyncStreamEncodingAndCompletion(t *testing.T) {
	for _, test := range []struct {
		encoding string
		gzip     bool
		count    int
	}{
		{"", false, 0}, {"", false, 201}, {"gzip", true, 201},
		{"gzip;q=0, *;q=1", false, 1}, {"br, *;q=0.5", true, 1},
	} {
		t.Run(test.encoding+"/"+strconv.Itoa(test.count), func(t *testing.T) {
			handler := newFullSyncHandler(func(ctx context.Context, emit func(GetImageResponse) error) error {
				for i := 0; i < test.count; i++ {
					if err := emit(GetImageResponse{GetImageRow: sqlc.GetImageRow{ID: int64(i + 1), Text: "line\nbreak and 中文"}, Tags: []Tag{}}); err != nil {
						return err
					}
				}
				return nil
			}, config.SyncConfig{})
			router := gin.New()
			router.GET("/sync/full", handler)
			req := httptest.NewRequest("GET", "/sync/full", nil)
			req.Header.Set("Accept-Encoding", test.encoding)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/x-ndjson" || rec.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("unexpected response: %d %v", rec.Code, rec.Header())
			}
			data := rec.Body.Bytes()
			if test.gzip {
				if rec.Header().Get("Content-Encoding") != "gzip" {
					t.Fatal("missing gzip header")
				}
				reader, err := gzip.NewReader(bytes.NewReader(data))
				if err != nil {
					t.Fatal(err)
				}
				data, err = io.ReadAll(reader)
				if err != nil {
					t.Fatal(err)
				}
				reader.Close()
			} else if rec.Header().Get("Content-Encoding") != "" {
				t.Fatal("gzip was explicitly disabled")
			}
			if images := readSyncExport(t, data); len(images) != test.count {
				t.Fatalf("exported %d images, want %d", len(images), test.count)
			}
		})
	}
}

func TestFullSyncFailureNeverCompletes(t *testing.T) {
	for _, afterWrite := range []bool{false, true} {
		router := gin.New()
		router.GET("/sync/full", newFullSyncHandler(func(ctx context.Context, emit func(GetImageResponse) error) error {
			if afterWrite {
				if err := emit(GetImageResponse{Tags: []Tag{}}); err != nil {
					return err
				}
			}
			// Also models a failed commit after all rows were emitted.
			return errors.New("database failed")
		}, config.SyncConfig{}))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest("GET", "/sync/full", nil))
		if strings.Contains(rec.Body.String(), `"type":"end"`) {
			t.Fatal("failed snapshot completed")
		}
		if !afterWrite && rec.Code != 500 {
			t.Fatalf("pre-stream error status = %d", rec.Code)
		}
	}
}

func TestFullSyncLimitsAndCancellation(t *testing.T) {
	entered := make(chan struct{})
	finished := make(chan struct{})
	router := gin.New()
	router.GET("/sync/full", newFullSyncHandler(func(ctx context.Context, emit func(GetImageResponse) error) error {
		close(entered)
		<-ctx.Done()
		return ctx.Err()
	}, config.SyncConfig{MaxConcurrent: 1, ExportTimeout: time.Minute}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		defer close(finished)
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/sync/full", nil).WithContext(ctx))
	}()
	<-entered
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/sync/full", nil))
	if rec.Code != 429 || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("concurrency limit: %d %v", rec.Code, rec.Header())
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("disconnect did not cancel snapshot")
	}

	router = gin.New()
	router.GET("/sync/full", newFullSyncHandler(func(ctx context.Context, emit func(GetImageResponse) error) error {
		<-ctx.Done()
		return ctx.Err()
	}, config.SyncConfig{ExportTimeout: time.Millisecond}))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/sync/full", nil))
	if rec.Code != 500 || strings.Contains(rec.Body.String(), `"type":"end"`) {
		t.Fatal("timed-out export completed")
	}
}

func TestFullSyncHTTPAuthenticationAndImageShape(t *testing.T) {
	pool, _ := testutil.Database(t)
	testutil.Seed(t, pool)
	db.Init(context.Background(), pool)
	t.Cleanup(func() { db.Init(context.Background(), nil) })
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	previous := config.GetConfig().JWT
	config.GetConfig().JWT = config.JWTConfig{PrivateKey: private, Issuer: "test", Audience: "test", SessionTTL: 1}
	t.Cleanup(func() { config.GetConfig().JWT = previous })
	token, err := utils.NewToken(2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO appkey(user_id,permission,key) VALUES(2,0,'sync-read-only')`); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.GET("/sync/full", middleware.Auth(true), NewFullSyncHandler(config.SyncConfig{}))
	router.GET("/image/:id", GetImage)
	detail := httptest.NewRecorder()
	router.ServeHTTP(detail, httptest.NewRequest("GET", "/image/1", nil))
	var want struct{ Data GetImageResponse }
	if err := json.Unmarshal(detail.Body.Bytes(), &want); err != nil || detail.Code != 200 {
		t.Fatal(detail.Body.String(), err)
	}
	for _, auth := range []string{"", "Key invalid", "Key sync-read-only", "Session " + token} {
		req := httptest.NewRequest("GET", "/sync/full", nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if auth == "" || auth == "Key invalid" {
			if rec.Code != 401 {
				t.Fatalf("anonymous/invalid auth accepted: %d", rec.Code)
			}
			continue
		}
		if rec.Code != http.StatusOK {
			t.Fatal(rec.Code, rec.Body.String())
		}
		images := readSyncExport(t, rec.Body.Bytes())
		if len(images) != 1 || !reflect.DeepEqual(images[0], want.Data) {
			t.Fatalf("export differs from image detail or includes unversioned image: %+v %+v", images, want.Data)
		}
		for _, field := range []string{"email", "passwordHash", "permission", "preference", "indexedVersion", "revision"} {
			if strings.Contains(rec.Body.String(), `"`+field+`":`) {
				t.Fatal("private or internal field exported:", field)
			}
		}
	}
}

func TestFullSyncUsesOneSnapshotAcrossBatches(t *testing.T) {
	pool, _ := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	db.Init(ctx, pool)
	t.Cleanup(func() { db.Init(ctx, nil) })
	if _, err := pool.Exec(ctx, `
INSERT INTO image(user_id,image_key,image_url)
SELECT 2,'batch-'||n,'https://example.com/'||n FROM generate_series(1,1002) n;
INSERT INTO version(image_id,user_id,text,rating)
SELECT id,2,'snapshot text','none' FROM image WHERE id>2;
UPDATE image SET current_version_id=v.id FROM version v WHERE v.image_id=image.id;
UPDATE image SET deleted_at=NOW() WHERE id=1004;
`); err != nil {
		t.Fatal(err)
	}
	images := []GetImageResponse{}
	err := streamCatalog(ctx, db.ReadSnapshot, func(image GetImageResponse) error {
		images = append(images, image)
		if len(images) == syncBatchSize {
			_, err := pool.Exec(ctx, `
WITH nv AS (INSERT INTO version(image_id,user_id,text,rating,version)
 VALUES(1002,2,'later text','violent',2) RETURNING id)
UPDATE image SET current_version_id=nv.id FROM nv WHERE image.id=1002;
INSERT INTO tag(name) VALUES('later tag');
INSERT INTO version_tag(version_id,tag_id)
SELECT current_version_id,t.id FROM image CROSS JOIN tag t WHERE image.id=1002 AND t.name='later tag';
UPDATE image SET deleted_at=NOW() WHERE id=1003;
UPDATE image SET deleted_at=NULL WHERE id=1004;
INSERT INTO image(user_id,image_key,image_url) VALUES(2,'new','https://example.com/new');
WITH nv AS (INSERT INTO version(image_id,user_id,text,rating)
 VALUES(1005,2,'new image','none') RETURNING id)
UPDATE image SET current_version_id=nv.id FROM nv WHERE image.id=1005;
`)
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1002 || images[len(images)-1].ID != 1003 {
		t.Fatalf("concurrent deletion/restoration/insertion changed snapshot: %d images", len(images))
	}
	image := images[len(images)-2]
	if image.ID != 1002 || image.Text != "snapshot text" || image.Version != 1 || len(image.Tags) != 0 {
		t.Fatalf("metadata and tags changed across batches: %+v", image)
	}
	if err := db.ReadSnapshot(ctx, func(q *sqlc.Queries) error { return q.CreateTags(ctx, []string{"forbidden"}) }); err == nil {
		t.Fatal("snapshot transaction allowed a write")
	}
}
