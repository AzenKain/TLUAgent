package services

import (
	"context"
	"testing"
	"time"

	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/crypto"
)

func newQuotaTestIdentity(t *testing.T) string {
	t.Helper()
	id, err := crypto.GenerateRandomHex(8)
	if err != nil {
		t.Fatalf("Failed to generate test identity: %v", err)
	}
	return id
}

func TestChatQuotaService_FixedWindowCounter(t *testing.T) {
	quota := &chatQuotaService{
		cache:      cache.NewTheineCache(10 << 20),
		guestLimit: 3,
		userLimit:  5,
	}
	ctx := context.Background()
	guest := "guest:" + newQuotaTestIdentity(t)
	user := "user:" + newQuotaTestIdentity(t)

	for i := 0; i < 3; i++ {
		if err := quota.Allow(ctx, guest); err != nil {
			t.Fatalf("Guest request %d should be allowed: %v", i+1, err)
		}
	}
	if err := quota.Allow(ctx, guest); err == nil {
		t.Fatalf("Guest request beyond guest limit must be rejected")
	}

	for i := 0; i < 5; i++ {
		if err := quota.Allow(ctx, user); err != nil {
			t.Fatalf("User request %d should be allowed: %v", i+1, err)
		}
	}
	if err := quota.Allow(ctx, user); err == nil {
		t.Fatalf("User request beyond user limit must be rejected")
	}

	otherGuest := "guest:" + newQuotaTestIdentity(t)
	if err := quota.Allow(ctx, otherGuest); err != nil {
		t.Fatalf("Different identity must have its own bucket: %v", err)
	}
}

func TestChatQuotaService_LimitZeroDisables(t *testing.T) {
	quota := &chatQuotaService{
		cache:      cache.NewTheineCache(10 << 20),
		guestLimit: 0,
		userLimit:  0,
	}
	for i := 0; i < 10; i++ {
		if err := quota.Allow(context.Background(), "guest:"+newQuotaTestIdentity(t)); err != nil {
			t.Fatalf("Zero limit should disable quota enforcement: %v", err)
		}
	}
}

func TestStreamLimiter_Caps(t *testing.T) {
	limiter := NewStreamLimiter(3, 2)

	releaseA, ok := limiter.TryAcquire("client:" + newQuotaTestIdentity(t))
	if !ok {
		t.Fatalf("First acquire should succeed")
	}
	defer releaseA()

	perClientKey := "client:" + newQuotaTestIdentity(t)
	releaseB1, ok := limiter.TryAcquire(perClientKey)
	if !ok {
		t.Fatalf("First per-client acquire should succeed")
	}
	releaseB2, ok := limiter.TryAcquire(perClientKey)
	if !ok {
		t.Fatalf("Second per-client acquire should succeed when per-client cap is 2")
	}
	if _, ok := limiter.TryAcquire(perClientKey); ok {
		t.Fatalf("Third per-client acquire should be rejected when per-client cap is 2")
	}
	releaseB1()
	releaseB2()

	releaseB3, ok := limiter.TryAcquire(perClientKey)
	if !ok {
		t.Fatalf("Release should free the per-client slot")
	}
	defer releaseB3()

	acquired := make(chan bool, 1)
	go func() {
		release, ok := limiter.TryAcquire("client:" + newQuotaTestIdentity(t))
		if ok {
			defer release()
		}
		acquired <- ok
	}()

	select {
	case ok := <-acquired:
		if !ok {
			t.Fatalf("Slot freed by releaseB1 should be available globally")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("TryAcquire blocked although a slot was free")
	}
}

func TestStreamLimiter_GlobalCapRejects(t *testing.T) {
	limiter := NewStreamLimiter(1, 0)

	keyA := "client:" + newQuotaTestIdentity(t)
	keyB := "client:" + newQuotaTestIdentity(t)

	releaseA, ok := limiter.TryAcquire(keyA)
	if !ok {
		t.Fatalf("First global acquire should succeed")
	}
	if _, ok := limiter.TryAcquire(keyB); ok {
		t.Fatalf("Second global acquire must be rejected when global cap reached")
	}
	releaseA()

	if _, ok := limiter.TryAcquire(keyB); !ok {
		t.Fatalf("Acquire should succeed after release")
	}
}
