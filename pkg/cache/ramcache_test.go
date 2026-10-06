package cache

import (
	"context"
	"testing"
	"time"
)

func TestRamCache_Basic(t *testing.T) {
	c := NewTheineCache(10 << 20)
	ctx := context.Background()

	key := "test:item:1"
	type Item struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	val := Item{Name: "TLU", Value: 1988}
	if err := c.Set(ctx, key, val, time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	var fetched Item
	if err := c.Get(ctx, key, &fetched); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if fetched.Name != "TLU" || fetched.Value != 1988 {
		t.Fatalf("Unexpected value: %+v", fetched)
	}

	if err := c.Del(ctx, key); err != nil {
		t.Fatalf("Del failed: %v", err)
	}
	if err := c.Get(ctx, key, &fetched); err == nil {
		t.Fatalf("Expected cache miss after Del, got nil err")
	}
}

func TestRamCache_DelByPattern(t *testing.T) {
	c := NewTheineCache(10 << 20)
	ctx := context.Background()

	_ = c.Set(ctx, "user:1", "alice", time.Minute)
	_ = c.Set(ctx, "user:2", "bob", time.Minute)
	_ = c.Set(ctx, "role:1", "admin", time.Minute)

	if err := c.DelByPattern(ctx, "user:*"); err != nil {
		t.Fatalf("DelByPattern failed: %v", err)
	}

	var res string
	if err := c.Get(ctx, "user:1", &res); err == nil {
		t.Fatalf("Expected user:1 to be deleted")
	}
	if err := c.Get(ctx, "role:1", &res); err != nil || res != "admin" {
		t.Fatalf("Expected role:1 to still exist")
	}
}

func TestRamCache_GetOrFetch(t *testing.T) {
	c := NewTheineCache(10 << 20)
	ctx := context.Background()

	callCount := 0
	fetcher := func() (any, error) {
		callCount++
		return "computed-result", nil
	}

	var out1 string
	if err := c.GetOrFetch(ctx, "computed:key", &out1, time.Minute, fetcher); err != nil {
		t.Fatalf("GetOrFetch failed: %v", err)
	}
	if out1 != "computed-result" || callCount != 1 {
		t.Fatalf("Unexpected result or callCount: %v, %v", out1, callCount)
	}

	var out2 string
	if err := c.GetOrFetch(ctx, "computed:key", &out2, time.Minute, fetcher); err != nil {
		t.Fatalf("GetOrFetch second call failed: %v", err)
	}
	if out2 != "computed-result" || callCount != 1 {
		t.Fatalf("Fetcher should not have been called twice, count=%d", callCount)
	}
}
