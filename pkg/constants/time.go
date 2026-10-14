package constants

import (
	"fmt"
	"time"
)

const (
	AccessTokenDuration  = 30 * time.Minute
	RefreshTokenDuration = 14 * 24 * time.Hour

	NormalCacheDuration = 10 * time.Minute
	ListCacheDuration   = 5 * time.Minute
	ShortCacheDuration  = 1 * time.Minute
)

// CurrentAcademicYear computes the Vietnamese academic year string dynamically based on the current calendar month.
func CurrentAcademicYear() string {
	now := time.Now()
	y := now.Year()
	if now.Month() < time.September {
		return fmt.Sprintf("%d-%d", y-1, y)
	}
	return fmt.Sprintf("%d-%d", y, y+1)
}
