package runner

import (
	"context"

	"github.com/ideamans/crux-cli/internal/cache"
	"github.com/ideamans/crux-cli/internal/crux"
)

// BQClient abstracts BigQuery operations.
type BQClient interface {
	QueryLatestMonth(ctx context.Context) (string, error)
	QueryDevice(ctx context.Context, origins []string, monthFrom, monthTo, device string) ([]crux.DeviceRow, error)
	Close()
}

// BQClientFactory creates a BQClient on demand (called lazily, only when a query is needed).
type BQClientFactory func(ctx context.Context) (BQClient, error)

// CacheStore abstracts local cache operations.
type CacheStore interface {
	GetLatestMonth() (*cache.LatestMonth, error)
	SaveLatestMonth(lm *cache.LatestMonth) error
	IsStale(lm *cache.LatestMonth) bool
	GetDevice(origin, monthFrom, monthTo string) ([]crux.DeviceRow, error)
	SaveDevice(origin, monthFrom, monthTo string, rows []crux.DeviceRow) error
}
