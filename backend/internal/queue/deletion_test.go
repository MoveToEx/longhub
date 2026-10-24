package queue

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/hibiken/asynq"
	"long/internal/deletion"
	"long/internal/sqlc"
	"long/internal/testutil"
	"sync"
	"testing"
	"time"
)

func TestIndexUsesCurrentStateAndSerializesModeration(t *testing.T) {
	pool, transact := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	q := sqlc.New(pool)
	service := deletion.Service{Transact: transact}
	id, err := service.Create(ctx, 1, 2, "duplicate")
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	indexDone, moderateDone := make(chan error, 1), make(chan error, 1)
	go func() {
		indexDone <- syncImageIndex(ctx, IndexArgs{ImageID: 1, VersionID: 1}, transact, func(_ context.Context, _ int64, doc *IndexPayloadItem) error {
			if doc == nil {
				return errors.New("unexpected removal")
			}
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	go func() { moderateDone <- service.Moderate(ctx, id, 1, "ACCEPT", "active") }()
	select {
	case err := <-moderateDone:
		t.Fatalf("moderation bypassed index lock: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	if err := <-indexDone; err != nil {
		t.Fatal(err)
	}
	if err := <-moderateDone; err != nil {
		t.Fatal(err)
	}
	pending, err := q.GetUnindexedImages(ctx)
	if err != nil || len(pending) != 1 || pending[0].ImageID != 1 || pending[0].VersionID != 0 {
		t.Fatalf("deleted image must remain eligible for reconciliation: %+v %v", pending, err)
	}
	deleted := 0
	write := func(_ context.Context, _ int64, doc *IndexPayloadItem) error {
		if doc != nil {
			return errors.New("stale job republished deleted image")
		}
		deleted++
		return nil
	}
	for _, version := range []int64{1, 0, 999} {
		if err := syncImageIndex(ctx, IndexArgs{ImageID: 1, VersionID: version}, transact, write); err != nil {
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
	if err := service.Moderate(ctx, id, 1, "REVOKE", "approved"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE version SET text='latest restored text' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	temporary := errors.New("search unavailable")
	if err := syncImageIndex(ctx, IndexArgs{ImageID: 1}, transact, func(context.Context, int64, *IndexPayloadItem) error { return temporary }); !errors.Is(err, temporary) {
		t.Fatal(err)
	}
	image, err := q.LockImageForDeletion(ctx, 1)
	if err != nil || image.IndexedVersion.Valid {
		t.Fatal("failed sync marked indexed", err)
	}
	if err := syncImageIndex(ctx, IndexArgs{ImageID: 1, VersionID: 999}, transact, func(_ context.Context, _ int64, doc *IndexPayloadItem) error {
		if doc == nil || doc.Text != "latest restored text" {
			return errors.New("stale metadata restored")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	image, _ = q.LockImageForDeletion(ctx, 1)
	if image.IndexedVersion.Int64 != image.CurrentVersionID.Int64 {
		t.Fatal("successful sync not committed")
	}
}

func TestOutboxRetriesAndDistinctApprovalSnapshots(t *testing.T) {
	pool, transact := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	service := deletion.Service{Transact: transact}
	if _, err := pool.Exec(ctx, `INSERT INTO webhook(user_id,label,endpoint,secret,event_types) VALUES(2,'delete','https://example.com/webhook','secret',4)`); err != nil {
		t.Fatal(err)
	}
	id, _ := service.Create(ctx, 1, 2, "duplicate")
	for _, step := range []struct {
		action string
		status sqlc.DeletionStatus
	}{{"ACCEPT", "active"}, {"REVOKE", "approved"}, {"ACCEPT", "revoked"}} {
		if err := service.Moderate(ctx, id, 1, step.action, step.status); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, "UPDATE version SET text='changed after approval' WHERE id=1; UPDATE public.user SET username='renamed admin' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	offline := errors.New("queue offline")
	if err := dispatchDeletionOutbox(ctx, transact, func(context.Context, *asynq.Task, ...asynq.Option) (*asynq.TaskInfo, error) { return nil, offline }); !errors.Is(err, offline) {
		t.Fatal(err)
	}
	var pending int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM deletion_outbox WHERE dispatched_at IS NULL").Scan(&pending)
	if pending != 6 {
		t.Fatal(pending)
	}
	var mu sync.Mutex
	tasks := map[string]*asynq.Task{}
	failAfterFirst := true
	enqueue := func(_ context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
		mu.Lock()
		defer mu.Unlock()
		key := ""
		for _, opt := range opts {
			if opt.Type() == asynq.TaskIDOpt {
				key = opt.Value().(string)
			}
		}
		if _, exists := tasks[key]; exists {
			return nil, asynq.ErrTaskIDConflict
		}
		if len(tasks) == 1 && failAfterFirst {
			failAfterFirst = false
			return nil, offline
		}
		tasks[key] = task
		return &asynq.TaskInfo{}, nil
	}
	if err := dispatchDeletionOutbox(ctx, transact, enqueue); !errors.Is(err, offline) {
		t.Fatal(err)
	}
	if err := dispatchDeletionOutbox(ctx, transact, enqueue); err != nil {
		t.Fatal(err)
	}
	events := map[string]bool{}
	names := map[string]int{}
	syncs := 0
	for _, task := range tasks {
		if task.Type() == TypeIndex {
			syncs++
			continue
		}
		var args InvokeArgs
		if err := json.Unmarshal(task.Payload(), &args); err != nil {
			t.Fatal(err)
		}
		var event deletion.Event
		_ = json.Unmarshal(args.Body, &event.Data)
		event.Initiator = args.Initiator
		if args.EventID == "" || events[args.EventID] || event.Data.Text != "original text" || event.Initiator.Username != "admin" {
			t.Fatal(args, event)
		}
		events[args.EventID] = true
		names[args.Event]++
	}
	if len(events) != 3 || syncs != 3 || names["delete"] != 2 || names["restore"] != 1 {
		t.Fatal(len(events), syncs, names)
	}
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM deletion_outbox WHERE dispatched_at IS NULL").Scan(&pending)
	if pending != 0 {
		t.Fatal(pending)
	}
	before := len(tasks)
	if err := dispatchDeletionOutbox(ctx, transact, enqueue); err != nil || len(tasks) != before {
		t.Fatal("dispatched twice", err)
	}
}
