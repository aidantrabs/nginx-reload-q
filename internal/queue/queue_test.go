package queue

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

func nopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSingleReload(t *testing.T) {
	var called int
	q := New(func(ctx context.Context) error {
		called++
		return nil
	}, nopLogger())

	q.Start(context.Background())
	q.Enqueue()
	q.Close()

	if called != 1 {
		t.Fatalf("expected 1 reload, got %d", called)
	}
}

func TestReloadFailureTracked(t *testing.T) {
	q := New(func(ctx context.Context) error {
		return fmt.Errorf("broken config")
	}, nopLogger())

	q.Start(context.Background())
	q.Enqueue()
	q.Close()

	stats := q.Stats()
	if stats.Failures != 1 {
		t.Fatalf("expected 1 failure, got %d", stats.Failures)
	}
	if stats.Reloads != 0 {
		t.Fatalf("expected 0 reloads, got %d", stats.Reloads)
	}
}

func TestDeduplication(t *testing.T) {
	// block the worker so we can pile up requests
	block := make(chan struct{})
	var called int

	q := New(func(ctx context.Context) error {
		<-block
		called++
		return nil
	}, nopLogger())

	q.Start(context.Background())

	// first enqueue goes through, worker picks it up and blocks
	q.Enqueue()
	time.Sleep(10 * time.Millisecond)

	// these should all deduplicate into one pending reload
	for i := 0; i < 20; i++ {
		q.Enqueue()
	}

	stats := q.Stats()
	if stats.Deduplicated < 19 {
		t.Fatalf("expected at least 19 deduplicated, got %d", stats.Deduplicated)
	}

	// unblock worker, let it finish both reloads
	close(block)
	q.Close()

	if called != 2 {
		t.Fatalf("expected 2 reloads (initial + 1 deduped batch), got %d", called)
	}
}

func TestStatsAfterSuccess(t *testing.T) {
	q := New(func(ctx context.Context) error {
		return nil
	}, nopLogger())

	q.Start(context.Background())
	q.Enqueue()
	q.Close()

	stats := q.Stats()
	if stats.Reloads != 1 {
		t.Fatalf("expected 1 reload, got %d", stats.Reloads)
	}
	if stats.LastReload == "" {
		t.Fatal("expected last_reload to be set")
	}
	if stats.Pending {
		t.Fatal("expected pending to be false")
	}
}

func TestConcurrentEnqueue(t *testing.T) {
	var mu sync.Mutex
	var called int

	q := New(func(ctx context.Context) error {
		mu.Lock()
		called++
		mu.Unlock()
		return nil
	}, nopLogger())

	q.Start(context.Background())

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.Enqueue()
		}()
	}
	wg.Wait()
	q.Close()

	// with dedup, we should get way fewer than 100 actual reloads
	mu.Lock()
	c := called
	mu.Unlock()

	if c < 1 || c > 100 {
		t.Fatalf("expected between 1 and 100 reloads, got %d", c)
	}

	stats := q.Stats()
	total := stats.Reloads + stats.Deduplicated
	// every enqueue either ran or was deduplicated
	if total < 100 {
		t.Fatalf("reloads (%d) + deduplicated (%d) should account for all 100 requests", stats.Reloads, stats.Deduplicated)
	}
}
