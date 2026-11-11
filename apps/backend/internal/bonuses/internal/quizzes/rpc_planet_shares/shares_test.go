package rpc_planet_shares_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/quizzes/rpc_planet_shares"
)

type fakePlanet struct {
	mu      sync.Mutex
	asked   int
	answer  *planetv1.GetTerritoriesResponse
	failing error
}

func (f *fakePlanet) GetTerritories(
	context.Context, *connect.Request[planetv1.GetTerritoriesRequest],
) (*connect.Response[planetv1.GetTerritoriesResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.asked++
	if f.failing != nil {
		return nil, f.failing
	}
	return connect.NewResponse(f.answer), nil
}

func (f *fakePlanet) times() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.asked
}

func (f *fakePlanet) set(answer *planetv1.GetTerritoriesResponse, failing error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.answer, f.failing = answer, failing
}

func running(t *testing.T, planet *fakePlanet) *rpc_planet_shares.Shares {
	t.Helper()

	shares := rpc_planet_shares.New(planet, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		shares.Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return shares
}

func TestAShareIsTheFlagsTilesOverTheMap(t *testing.T) {
	planet := &fakePlanet{answer: &planetv1.GetTerritoriesResponse{Tiles: 200, Territories: []*planetv1.Territory{
		{CountryId: "fr", Tiles: 50}, {CountryId: "jp", Tiles: 10},
	}}}

	shares := running(t, planet)

	assert.Eventually(t, func() bool { return shares.Share("fr") == 0.25 }, 5*time.Second, 5*time.Millisecond)
	assert.InDelta(t, 0.05, shares.Share("jp"), 1e-9)
	assert.Zero(t, shares.Share("de"), "a flag that holds nothing")
}

func TestBeforePlanetAnswersEveryShareIsZero(t *testing.T) {
	shares := rpc_planet_shares.New(&fakePlanet{}, time.Hour)

	assert.Zero(t, shares.Share("fr"))
}

func TestAFailedAskKeepsTheLastShares(t *testing.T) {
	planet := &fakePlanet{answer: &planetv1.GetTerritoriesResponse{Tiles: 100, Territories: []*planetv1.Territory{
		{CountryId: "fr", Tiles: 50},
	}}}
	shares := running(t, planet)
	assert.Eventually(t, func() bool { return shares.Share("fr") == 0.5 }, 5*time.Second, 5*time.Millisecond)

	planet.set(nil, errors.New("planet is gone"))
	asked := planet.times()
	assert.Eventually(t, func() bool { return planet.times() > asked+1 }, 5*time.Second, 5*time.Millisecond)

	assert.InDelta(t, 0.5, shares.Share("fr"), 1e-9)
}
