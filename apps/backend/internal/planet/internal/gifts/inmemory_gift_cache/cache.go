package inmemory_gift_cache

import (
	"context"
	"errors"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/gifts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type given struct {
	tag    tempo.GiftTag
	holder bonuses.Holder
}

func New(storage gifts.Storage) *Cache {
	return &Cache{storage: storage, given: cpcolls.NewSet[given]()}
}

// Every click asks during a finale, and only an account's first may wait on the store.
type Cache struct {
	storage gifts.Storage

	mu    sync.RWMutex
	given *cpcolls.Set[given]
}

var _ gifts.Storage = (*Cache)(nil)

func (c *Cache) Give(ctx context.Context, tag tempo.GiftTag, holder bonuses.Holder) error {
	key := given{tag: tag, holder: holder}

	c.mu.RLock()
	known := c.given.Contains(key)
	c.mu.RUnlock()
	if known {
		return gifts.ErrGiven
	}

	err := c.storage.Give(ctx, tag, holder)
	if err != nil && !errors.Is(err, gifts.ErrGiven) {
		return err //nolint:wrapcheck // the store named it.
	}

	c.mu.Lock()
	c.given.Add(key)
	c.mu.Unlock()

	return err //nolint:wrapcheck // ErrGiven, as the store answered it.
}
