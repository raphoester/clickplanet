package caching_fronts_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/rpc_planet_fronts/caching_fronts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const ttl = 30 * time.Second

type countingFronts struct {
	tiles uint64
	err   error
	asked []accounts.AccountID
}

func (c *countingFronts) Fronts(_ context.Context, account accounts.AccountID) (*progressionv1.GetFrontsResponse, error) {
	c.asked = append(c.asked, account)
	if c.err != nil {
		return nil, c.err
	}
	return &progressionv1.GetFrontsResponse{PlaysFor: []*progressionv1.CountryTiles{{CountryId: "fr", Tiles: c.tiles}}}, nil
}

func tilesOf(t *testing.T, cache *caching_fronts.Cache, account accounts.AccountID) uint64 {
	t.Helper()

	fronts, err := cache.Fronts(t.Context(), account)
	require.NoError(t, err)
	return fronts.GetPlaysFor()[0].GetTiles()
}

func TestAnAnswerIsKeptUntilItsTimeIsUp(t *testing.T) {
	inner := &countingFronts{tiles: 1}
	clock := cptime.NewFixedClock(time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC))
	cache := caching_fronts.New(inner, ttl, clock)

	assert.Equal(t, uint64(1), tilesOf(t, cache, accounts.AccountID{15: 1}))
	inner.tiles = 2
	clock.Advance(ttl - time.Second)
	assert.Equal(t, uint64(1), tilesOf(t, cache, accounts.AccountID{15: 1}))

	clock.Advance(time.Second)
	assert.Equal(t, uint64(2), tilesOf(t, cache, accounts.AccountID{15: 1}))
	assert.Len(t, inner.asked, 2)
}

func TestEachAccountIsKeptApart(t *testing.T) {
	inner := &countingFronts{tiles: 1}
	cache := caching_fronts.New(inner, ttl, cptime.NewFixedClock(time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)))

	tilesOf(t, cache, accounts.AccountID{15: 1})
	tilesOf(t, cache, accounts.AccountID{15: 2})
	tilesOf(t, cache, accounts.AccountID{15: 1})

	assert.Equal(t, []accounts.AccountID{{15: 1}, {15: 2}}, inner.asked)
}

func TestAFailureIsNotKept(t *testing.T) {
	inner := &countingFronts{err: errors.New("planet is down")}
	cache := caching_fronts.New(inner, ttl, cptime.NewFixedClock(time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)))

	_, err := cache.Fronts(t.Context(), accounts.AccountID{15: 1})
	require.ErrorIs(t, err, inner.err)
	inner.err = nil
	inner.tiles = 3

	fronts, err := cache.Fronts(t.Context(), accounts.AccountID{15: 1})
	require.NoError(t, err)
	assert.True(t, proto.Equal(&progressionv1.GetFrontsResponse{PlaysFor: []*progressionv1.CountryTiles{{CountryId: "fr", Tiles: 3}}}, fronts))
}

func TestAFullCacheMakesRoomOnlyFromAnswersWhoseTimeIsUp(t *testing.T) {
	inner := &countingFronts{tiles: 1}
	clock := cptime.NewFixedClock(time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC))
	cache := caching_fronts.New(inner, ttl, clock)
	for i := range 4096 {
		tilesOf(t, cache, accounts.AccountID{14: byte(i >> 8), 15: byte(i)})
	}

	tilesOf(t, cache, accounts.AccountID{0: 1})
	tilesOf(t, cache, accounts.AccountID{0: 1})
	assert.Len(t, inner.asked, 4096+2, "a full cache keeps nothing new")

	clock.Advance(ttl)
	tilesOf(t, cache, accounts.AccountID{0: 1})
	tilesOf(t, cache, accounts.AccountID{0: 1})
	assert.Len(t, inner.asked, 4096+3)
}
