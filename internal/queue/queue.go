package queue

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

type ReloadFunc func(ctx context.Context) error

type Stats struct {
	Reloads     int64  `json:"reloads"`
	Failures    int64  `json:"failures"`
	LastReload  string `json:"last_reload,omitempty"`
	Pending     bool   `json:"pending"`
	Deduplicated int64 `json:"deduplicated"`
}

type Queue struct {
	jobs         chan struct{}
	log          *slog.Logger
	wg           sync.WaitGroup
	reload       ReloadFunc
	pending      atomic.Bool
	reloads      atomic.Int64
	failures     atomic.Int64
	deduplicated atomic.Int64
	lastReload   atomic.Value
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
		q.deduplicated.Add(1)
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

func (q *Queue) Stats() Stats {
	var last string
	if v := q.lastReload.Load(); v != nil {
		last = v.(string)
	}

	return Stats{
		Reloads:      q.reloads.Load(),
		Failures:     q.failures.Load(),
		LastReload:   last,
		Pending:      q.pending.Load(),
		Deduplicated: q.deduplicated.Load(),
	}
}

func (q *Queue) worker(ctx context.Context) {
	defer q.wg.Done()

	for range q.jobs {
		q.log.Info("worker processing reload")
		q.pending.Store(false)

		if err := q.reload(ctx); err != nil {
			q.failures.Add(1)
			q.log.Error("reload failed", "err", err)
		} else {
			q.reloads.Add(1)
			q.lastReload.Store(time.Now().UTC().Format(time.RFC3339))
			q.log.Info("worker reload complete")
		}
	}

	q.log.Info("worker stopped")
}
