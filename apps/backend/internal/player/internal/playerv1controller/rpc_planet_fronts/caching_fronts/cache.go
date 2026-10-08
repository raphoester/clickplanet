package caching_fronts

import (
	"context"
	"sync"
	"time"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Fronts interface {
	Fronts(ctx context.Context, account cpsession.AccountID) (*playerv1.GetFrontsResponse, error)
}

const most = 4096

type entry struct {
	fronts *playerv1.GetFrontsResponse
	until  time.Time
}

type Cache struct {
	inner Fronts
	ttl   time.Duration
	clock cptime.Clock

	mu   sync.Mutex
	kept map[cpsession.AccountID]entry
}

func New(inner Fronts, ttl time.Duration, clock cptime.Clock) *Cache {
	return &Cache{inner: inner, ttl: ttl, clock: clock, kept: map[cpsession.AccountID]entry{}}
}

func (c *Cache) Fronts(ctx context.Context, account cpsession.AccountID) (*playerv1.GetFrontsResponse, error) {
	if fronts, ok := c.keptFronts(account); ok {
		return fronts, nil
	}

	fronts, err := c.inner.Fronts(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the reader named it.
	}
	c.keep(account, fronts)
	return fronts, nil
}

func (c *Cache) keptFronts(account cpsession.AccountID) (*playerv1.GetFrontsResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	kept, ok := c.kept[account]
	if !ok || !c.clock.Now().Before(kept.until) {
		return nil, false
	}
	return kept.fronts, true
}

func (c *Cache) keep(account cpsession.AccountID, fronts *playerv1.GetFrontsResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.clock.Now()
	if len(c.kept) >= most {
		for kept, e := range c.kept {
			if !now.Before(e.until) {
				delete(c.kept, kept)
			}
		}
	}
	if len(c.kept) >= most {
		return
	}
	c.kept[account] = entry{fronts: fronts, until: now.Add(c.ttl)}
}
