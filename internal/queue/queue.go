package queue

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
)

type ReloadFunc func(ctx context.Context) error

type Queue struct {
	jobs    chan struct{}
	log     *slog.Logger
	wg      sync.WaitGroup
	reload  ReloadFunc
	pending atomic.Bool
}

func New(reload ReloadFunc, log *slog.Logger) *Queue {
	return &Queue{
		jobs:   make(chan struct{}, 1),
		log:    log,
		reload: reload,
	}
}

func (q *Queue) Start(ctx context.Context) {
	q.wg.Add(1)
	go q.worker(ctx)
}

// enqueues a reload, collapses duplicates if one is already pending
func (q *Queue) Enqueue() bool {
	if !q.pending.CompareAndSwap(false, true) {
		q.log.Info("reload already pending, deduplicating")
		return true
	}

	q.jobs <- struct{}{}
	q.log.Info("reload enqueued")

	return true
}

// signals the worker to stop and waits for it to drain
func (q *Queue) Close() {
	close(q.jobs)
	q.wg.Wait()
}

func (q *Queue) Pending() bool {
	return q.pending.Load()
}

func (q *Queue) worker(ctx context.Context) {
	defer q.wg.Done()

	for range q.jobs {
		q.log.Info("worker processing reload")
		q.pending.Store(false)

		if err := q.reload(ctx); err != nil {
			q.log.Error("reload failed", "err", err)
		} else {
			q.log.Info("worker reload complete")
		}
	}

	q.log.Info("worker stopped")
}
