package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type mockLifecycle struct {
	queuedCount    int32
	runningCount   int32
	completedCount int32
	failedCount    int32
}

func (m *mockLifecycle) Queued(ctx context.Context, job Job) error {
	atomic.AddInt32(&m.queuedCount, 1)
	return nil
}

func (m *mockLifecycle) Running(ctx context.Context, job Job) error {
	atomic.AddInt32(&m.runningCount, 1)
	return nil
}

func (m *mockLifecycle) Completed(ctx context.Context, job Job) error {
	atomic.AddInt32(&m.completedCount, 1)
	return nil
}

func (m *mockLifecycle) Failed(ctx context.Context, job Job, err error) error {
	atomic.AddInt32(&m.failedCount, 1)
	return nil
}

func TestQueue_ProcessSuccess(t *testing.T) {
	q := NewQueue(2, 100)
	lc := &mockLifecycle{}
	q.SetLifecycle(lc)

	var executed int32
	q.RegisterHandler("task.success", func(ctx context.Context, jobID string, payload string) error {
		atomic.AddInt32(&executed, 1)
		return nil
	})

	q.Start()
	defer q.Stop()

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		err := q.Enqueue(ctx, Job{
			ID:      "job-test",
			Type:    "task.success",
			Payload: "payload",
		})
		if err != nil {
			t.Fatalf("Enqueue failed: %v", err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&executed) != 5 {
		t.Fatalf("Expected 5 jobs executed, got: %d", atomic.LoadInt32(&executed))
	}
	if atomic.LoadInt32(&lc.completedCount) != 5 {
		t.Fatalf("Expected 5 completed events, got: %d", atomic.LoadInt32(&lc.completedCount))
	}
}

func TestQueue_ProcessRetryAndFail(t *testing.T) {
	q := NewQueue(1, 10)
	lc := &mockLifecycle{}
	q.SetLifecycle(lc)

	var attempts int32
	q.RegisterHandler("task.fail", func(ctx context.Context, jobID string, payload string) error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("simulated error")
	})

	q.Start()
	defer q.Stop()

	ctx := context.Background()
	_ = q.Enqueue(ctx, Job{
		ID:      "job-fail",
		Type:    "task.fail",
		Payload: "payload",
	})

	time.Sleep(2 * time.Second)

	if atomic.LoadInt32(&attempts) != 3 {
		t.Fatalf("Expected 3 retry attempts, got: %d", atomic.LoadInt32(&attempts))
	}
	if atomic.LoadInt32(&lc.failedCount) != 1 {
		t.Fatalf("Expected 1 failed event, got: %d", atomic.LoadInt32(&lc.failedCount))
	}
}
