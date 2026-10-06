package services

import "sync"

// StreamLimiter bounds concurrent SSE streams server-wide and per client.
type StreamLimiter struct {
	mu           sync.Mutex
	global       chan struct{}
	maxPerClient int
	perClient    map[string]int
}

// NewStreamLimiter creates a limiter with the given global and per-client caps.
func NewStreamLimiter(maxGlobal, maxPerClient int) *StreamLimiter {
	if maxGlobal <= 0 {
		maxGlobal = 1
	}
	return &StreamLimiter{
		global:       make(chan struct{}, maxGlobal),
		maxPerClient: maxPerClient,
		perClient:    make(map[string]int),
	}
}

// TryAcquire reserves one stream slot and returns a release function on success.
func (l *StreamLimiter) TryAcquire(clientKey string) (func(), bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.maxPerClient > 0 && l.perClient[clientKey] >= l.maxPerClient {
		return nil, false
	}
	select {
	case l.global <- struct{}{}:
	default:
		return nil, false
	}

	if l.maxPerClient > 0 {
		l.perClient[clientKey]++
	}

	released := false
	return func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		if released {
			return
		}
		released = true
		if l.maxPerClient > 0 {
			l.perClient[clientKey]--
			if l.perClient[clientKey] <= 0 {
				delete(l.perClient, clientKey)
			}
		}
		<-l.global
	}, true
}
