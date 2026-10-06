package constants

import "time"

const (
	AccessTokenDuration  = 30 * time.Minute
	RefreshTokenDuration = 14 * 24 * time.Hour

	NormalCacheDuration = 10 * time.Minute
	ListCacheDuration   = 5 * time.Minute
	ShortCacheDuration  = 1 * time.Minute
)
