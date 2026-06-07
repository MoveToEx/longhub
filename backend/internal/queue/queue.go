package queue

import (
	"context"
	"time"

	"long/internal/config"

	"github.com/hibiken/asynq"
)

const (
	TypeHash  = "hash"
	TypeIndex = "ms-index"
)

var (
	client    *asynq.Client
	server    *asynq.Server
	scheduler *asynq.Scheduler
)

func redisOpt() asynq.RedisConnOpt {
	return asynq.RedisClientOpt{Addr: config.GetConfig().ValkeyAddr}
}

func Init() error {
	opt := redisOpt()
	client = asynq.NewClient(opt)
	server = asynq.NewServer(opt, asynq.Config{
		Concurrency: 100,
	})
	scheduler = asynq.NewScheduler(opt, &asynq.SchedulerOpts{
		Location: time.Local,
	})

	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeHash, HandleHashTask)
	mux.HandleFunc(TypeIndex, HandleIndexTask)

	if err := server.Start(mux); err != nil {
		_ = client.Close()
		return err
	}

	if _, err := scheduler.Register("@every 3s", NewIndexTask()); err != nil {
		shutdown()
		return err
	}

	if err := scheduler.Start(); err != nil {
		shutdown()
		return err
	}

	_, err := client.EnqueueContext(context.Background(), NewIndexTask())
	if err != nil {
		shutdown()
	}
	return err
}

func shutdown() error {
	if scheduler != nil {
		scheduler.Shutdown()
	}
	if server != nil {
		server.Shutdown()
	}
	if client != nil {
		return client.Close()
	}
	return nil
}

func Stop() error {
	return shutdown()
}
