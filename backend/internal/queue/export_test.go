package queue

import (
	"context"
	"testing"

	"github.com/hibiken/asynq"
)

// Expose queue internals to integration tests without introducing an import cycle.
var SyncImageIndexForTest = syncImageIndex

type testTaskClient struct {
	enqueue func(context.Context, *asynq.Task, ...asynq.Option) (*asynq.TaskInfo, error)
}

func (c testTaskClient) EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	return c.enqueue(ctx, task, opts...)
}

func (testTaskClient) Close() error { return nil }

func SetEnqueueForTest(t *testing.T, enqueue func(context.Context, *asynq.Task, ...asynq.Option) (*asynq.TaskInfo, error)) {
	t.Helper()
	previous := client
	client = testTaskClient{enqueue: enqueue}
	t.Cleanup(func() { client = previous })
}
