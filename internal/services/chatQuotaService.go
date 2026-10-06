package services

import (
	"context"
	"strings"
	"sync"
	"time"

	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/config"
)

// ChatQuotaService enforces a per-identity daily fixed-window chat allowance.
type ChatQuotaService interface {
	Allow(ctx context.Context, identity string) error
}

type chatQuotaService struct {
	cache      cache.Cache
	guestLimit int
	userLimit  int
	mu         sync.Mutex
}

// NewChatQuotaService creates a chat quota service backed by the shared RAM cache.
func NewChatQuotaService(ramCache cache.Cache) ChatQuotaService {
	return &chatQuotaService{
		cache:      ramCache,
		guestLimit: config.GetIntConfigWithDefault("CHAT_GUEST_DAILY_QUOTA", 20),
		userLimit:  config.GetIntConfigWithDefault("CHAT_USER_DAILY_QUOTA", 200),
	}
}

func (s *chatQuotaService) Allow(ctx context.Context, identity string) error {
	limit := s.userLimit
	if strings.HasPrefix(identity, "guest:") {
		limit = s.guestLimit
	}
	if limit <= 0 {
		return nil
	}

	now := time.Now().UTC()
	key := cache.BuildKey("chat_quota", now.Format("2006-01-02"), identity)
	ttl := time.Until(time.Date(now.Year(), now.Month(), now.Day(), 24, 0, 0, 0, time.UTC))

	s.mu.Lock()
	defer s.mu.Unlock()

	var count int
	if err := s.cache.Get(ctx, key, &count); err != nil {
		count = 0
	}
	if count >= limit {
		return apperrors.New(apperrors.ErrTooManyRequests, "Daily chat quota exceeded. Please try again tomorrow or sign in with a full account.")
	}
	count++
	return s.cache.Set(ctx, key, count, ttl)
}
