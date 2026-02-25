package queue

import (
	"log/slog"
	"sync"
	"time"
)

type Queue struct {
	jobs chan struct{}
	log  *slog.Logger
	wg   sync.WaitGroup
}

func New(size int, log *slog.Logger) *Queue {
	return &Queue{
		jobs: make(chan struct{}, size),
		log:  log,
	}
}

// launches the worker goroutine
func (q *Queue) Start() {
	q.wg.Add(1)
	go q.worker()
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

func (q *Queue) worker() {
	defer q.wg.Done()

	for range q.jobs {
		q.log.Info("worker processing reload")

		// simulated reload, replaced later
		time.Sleep(500 * time.Millisecond)

		q.log.Info("worker reload complete")
	}

	q.log.Info("worker stopped")
}
