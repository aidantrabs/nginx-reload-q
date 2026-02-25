package queue

import (
	"context"
	"log/slog"
	"sync"
)

type ReloadFunc func(ctx context.Context) error

type Queue struct {
	jobs   chan struct{}
	log    *slog.Logger
	wg     sync.WaitGroup
	reload ReloadFunc
}

func New(size int, reload ReloadFunc, log *slog.Logger) *Queue {
	return &Queue{
		jobs:   make(chan struct{}, size),
		log:    log,
		reload: reload,
	}
}

func (q *Queue) Start(ctx context.Context) {
	q.wg.Add(1)
	go q.worker(ctx)
}

// adds a reload job, returns false if queue is full
func (q *Queue) Enqueue() bool {
	select {
	case q.jobs <- struct{}{}:
		q.log.Info("reload enqueued", "depth", len(q.jobs))
		return true
	default:
		q.log.Warn("queue full, dropping request")
		return false
	}
}

// signals the worker to stop and waits for it to drain
func (q *Queue) Close() {
	close(q.jobs)
	q.wg.Wait()
}

// returns the current number of pending jobs
func (q *Queue) Depth() int {
	return len(q.jobs)
}

func (q *Queue) worker(ctx context.Context) {
	defer q.wg.Done()

	for range q.jobs {
		q.log.Info("worker processing reload")

		if err := q.reload(ctx); err != nil {
			q.log.Error("reload failed", "err", err)
		} else {
			q.log.Info("worker reload complete")
		}
	}

	q.log.Info("worker stopped")
}
