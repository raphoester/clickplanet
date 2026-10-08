package inmemory_gift_cache_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/gifts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/gifts/inmemory_gift_cache"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/gifts/inmemory_gift_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, &gifts.StorageContractSuite{NewStorage: func() gifts.Storage {
		return inmemory_gift_cache.New(inmemory_gift_storage.New())
	}})
}

const ada bonuses.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

type countingStorage struct {
	gifts.Storage
	asked int
}

func (c *countingStorage) Give(ctx context.Context, tag tempo.GiftTag, holder bonuses.Holder) error {
	c.asked++
	return c.Storage.Give(ctx, tag, holder) //nolint:wrapcheck // a test double passes it on.
}

func TestAGiftKnownToBeGivenIsNotAskedAgain(t *testing.T) {
	store := &countingStorage{Storage: inmemory_gift_storage.New()}
	cache := inmemory_gift_cache.New(store)

	require.NoError(t, cache.Give(t.Context(), "finale-0", ada))
	require.ErrorIs(t, cache.Give(t.Context(), "finale-0", ada), gifts.ErrGiven)
	require.ErrorIs(t, cache.Give(t.Context(), "finale-0", ada), gifts.ErrGiven)

	require.Equal(t, 1, store.asked)
}

func TestAGiftGivenBeforeARestartIsLearntFromTheStore(t *testing.T) {
	store := inmemory_gift_storage.New()
	require.NoError(t, inmemory_gift_cache.New(store).Give(t.Context(), "finale-0", ada))

	counting := &countingStorage{Storage: store}
	restarted := inmemory_gift_cache.New(counting)

	require.ErrorIs(t, restarted.Give(t.Context(), "finale-0", ada), gifts.ErrGiven)
	require.ErrorIs(t, restarted.Give(t.Context(), "finale-0", ada), gifts.ErrGiven)
	require.Equal(t, 1, counting.asked)
}

func TestAFailureIsNotRemembered(t *testing.T) {
	store := inmemory_gift_storage.New()
	failure := errors.New("postgres is down")
	store.FailWith(failure)
	cache := inmemory_gift_cache.New(store)

	require.ErrorIs(t, cache.Give(t.Context(), "finale-0", ada), failure)

	store.FailWith(nil)
	require.NoError(t, cache.Give(t.Context(), "finale-0", ada), "the gift is still owed")
}
