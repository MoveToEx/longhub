package queue_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"long/internal/admin"
	"long/internal/db"
	"long/internal/queue"
	"long/internal/sqlc"
	"long/internal/testutil"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestIndexUsesCurrentStateAndSerializesModeration(t *testing.T) {
	pool, transact := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db.Init(ctx, pool)
	t.Cleanup(func() { db.Init(ctx, nil) })
	q := sqlc.New(pool)
	id, err := q.CreateDeletionRequest(ctx, sqlc.CreateDeletionRequestParams{ImageID: 1, UserID: 2, Reason: "duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	releaseIndex := sync.OnceFunc(func() { close(release) })
	var workers sync.WaitGroup
	defer func() {
		releaseIndex()
		cancel()
		finished := make(chan struct{})
		go func() { workers.Wait(); close(finished) }()
		select {
		case <-finished:
		case <-time.After(5 * time.Second):
			t.Error("test workers did not stop after cancellation")
		}
	}()
	indexDone, moderateDone := make(chan error, 1), make(chan error, 1)
	workers.Add(1)
	go func() {
		defer workers.Done()
		indexDone <- queue.SyncImageIndexForTest(ctx, queue.IndexArgs{ImageID: 1, VersionID: 1}, transact, func(_ context.Context, _ int64, doc *queue.IndexPayloadItem) error {
			if doc == nil {
				return errors.New("unexpected removal")
			}
			close(entered)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	select {
	case <-entered:
	case err := <-indexDone:
		t.Fatalf("indexing ended before taking the lock: %v", err)
	case <-ctx.Done():
		t.Fatal("indexing never entered the locked write", ctx.Err())
	}
	workers.Add(1)
	go func() { defer workers.Done(); moderateDone <- moderateDeletion(ctx, id, "ACCEPT", "active") }()
	// The isolated database has only these two operations. Observe actual PostgreSQL
	// lock contention before releasing indexing, instead of relying on scheduling delays.
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	blocked := false
	for !blocked {
		select {
		case err := <-moderateDone:
			t.Fatalf("moderation bypassed the index lock: %v", err)
		case <-ctx.Done():
			t.Fatal("moderation never reached the contested lock", ctx.Err())
		case <-ticker.C:
			if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND cardinality(pg_blocking_pids(pid))>0)").Scan(&blocked); err != nil {
				t.Fatal(err)
			}
		}
	}
	releaseIndex()
	await := func(done <-chan error) {
		t.Helper()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-ctx.Done():
			t.Fatal("operation did not complete after releasing the lock", ctx.Err())
		}
	}
	await(indexDone)
	await(moderateDone)
	pending, err := q.GetUnindexedImages(ctx)
	if err != nil || len(pending) != 1 || pending[0].ImageID != 1 || pending[0].VersionID != 0 {
		t.Fatalf("deleted image must remain eligible for reconciliation: %+v %v", pending, err)
	}
	deleted := 0
	write := func(_ context.Context, _ int64, doc *queue.IndexPayloadItem) error {
		if doc != nil {
			return errors.New("stale job republished deleted image")
		}
		deleted++
		return nil
	}
	for _, version := range []int64{1, 0, 999} {
		if err := queue.SyncImageIndexForTest(ctx, queue.IndexArgs{ImageID: 1, VersionID: version}, transact, write); err != nil {
			t.Fatal(err)
		}
	}
	if deleted != 3 {
		t.Fatal(deleted)
	}
	pending, err = q.GetUnindexedImages(ctx)
	if err != nil || len(pending) != 0 {
		t.Fatalf("acknowledged removal remained pending: %+v %v", pending, err)
	}
	if err := moderateDeletion(ctx, id, "REOPEN", "approved"); err == nil {
		t.Fatal("approved deletion was reopened")
	}
	temporary := errors.New("search unavailable")
	if err := queue.SyncImageIndexForTest(ctx, queue.IndexArgs{ImageID: 1}, transact, func(context.Context, int64, *queue.IndexPayloadItem) error { return temporary }); !errors.Is(err, temporary) {
		t.Fatal(err)
	}
	image, err := q.LockImageForDeletion(ctx, 1)
	if err != nil || !image.DeletedAt.Valid || !image.IndexedVersion.Valid || image.IndexedVersion.Int64 != 0 {
		t.Fatal("failed sync changed the deletion tombstone", image, err)
	}
}

func TestDeletionEnqueuesAfterCommitAndSurvivesQueueFailure(t *testing.T) {
	pool, _ := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	db.Init(ctx, pool)
	t.Cleanup(func() { db.Init(ctx, nil) })
	q := sqlc.New(pool)
	id, err := q.CreateDeletionRequest(ctx, sqlc.CreateDeletionRequestParams{ImageID: 1, UserID: 2, Reason: "duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	var tasks []*asynq.Task
	offline := errors.New("queue offline")
	queue.SetEnqueueForTest(t, func(_ context.Context, task *asynq.Task, _ ...asynq.Option) (*asynq.TaskInfo, error) {
		image, err := q.GetImageDetails(ctx, 1)
		if err != nil || !image.DeletedAt.Valid || image.ActiveDeletionID.Int64 != id {
			t.Fatal("task enqueued before deletion committed", image, err)
		}
		tasks = append(tasks, task)
		return nil, offline
	})
	if err := moderateDeletion(ctx, id, "ACCEPT", "active"); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 || tasks[0].Type() != queue.TypeIndex || tasks[1].Type() != queue.TypeDispatch {
		t.Fatal("both queue operations must be attempted", tasks)
	}
	var args queue.DispatchArgs
	if err := json.Unmarshal(tasks[1].Payload(), &args); err != nil {
		t.Fatal(err)
	}
	if args.Deletion == nil || args.Deletion.Data.Text != "original text" || args.Deletion.Initiator.Username != "admin" || args.Deletion.Data.DeletionEventID == "" {
		t.Fatal("deletion snapshot was not queued", args)
	}
	pending, err := q.GetUnindexedImages(ctx)
	if err != nil || len(pending) != 1 || pending[0].ImageID != 1 || pending[0].VersionID != 0 {
		t.Fatal("queue failure prevented index reconciliation", pending, err)
	}
	if err := moderateDeletion(ctx, id, "ACCEPT", "approved"); err == nil || len(tasks) != 2 {
		t.Fatal("conflicting approval enqueued more work", err, len(tasks))
	}
}

func moderateDeletion(ctx context.Context, id int64, action string, expected sqlc.DeletionStatus) error {
	body, err := json.Marshal(map[string]any{
		"query": `mutation($id: ID!, $action: DeletionAction!, $expectedStatus: DeletionStatus!) {
			moderateDeletionRequest(id: $id, action: $action, expectedStatus: $expectedStatus) { id }
		}`,
		"variables": map[string]any{"id": id, "action": action, "expectedStatus": strings.ToUpper(string(expected))},
	})
	if err != nil {
		return err
	}
	router := gin.New()
	router.POST("/admin/graphql", func(c *gin.Context) { c.Set("UserID", int64(1)) }, admin.GraphQL)
	request := httptest.NewRequest("POST", "/admin/graphql", bytes.NewReader(body)).WithContext(ctx)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != 200 || strings.Contains(response.Body.String(), `"errors"`) {
		return fmt.Errorf("moderate deletion: HTTP %d: %s", response.Code, response.Body.String())
	}
	return nil
}
