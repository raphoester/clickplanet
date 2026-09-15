package inmemory_tile_storage_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
)

type save struct {
	tiles  []uint32
	owners []string
}

type fakePersistence struct {
	mu      sync.Mutex
	rows    map[uint32]string
	saves   []save
	failing error
}

func newFakePersistence() *fakePersistence {
	return &fakePersistence{rows: map[uint32]string{}}
}

func (f *fakePersistence) Load(_ context.Context, visit func(tile uint32, owner string)) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failing != nil {
		return f.failing
	}
	for tile, owner := range f.rows {
		visit(tile, owner)
	}
	return nil
}

func (f *fakePersistence) Save(_ context.Context, tiles []uint32, owners []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failing != nil {
		return f.failing
	}
	f.saves = append(f.saves, save{tiles: slices.Clone(tiles), owners: slices.Clone(owners)})
	for i, tile := range tiles {
		if owners[i] == "" {
			delete(f.rows, tile)
			continue
		}
		f.rows[tile] = owners[i]
	}
	return nil
}

func (f *fakePersistence) stored() map[uint32]string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return maps.Clone(f.rows)
}

func (f *fakePersistence) fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.failing = err
}

func (s *testSuite) TestLoadRestoresTheMapAndTheShares() {
	persistence := newFakePersistence()
	persistence.rows = map[uint32]string{1: "bg", 2: "bg", 3: "fr", maxIndex: "de"}

	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Load(context.Background()))

	state, err := stateBatch(storage, 0, maxIndex)
	s.Require().NoError(err)
	s.Equal(persistence.rows, state)
	s.InDelta(2.0/maxIndex, storage.Share("bg"), 1e-12)
	s.InDelta(1.0/maxIndex, storage.Share("fr"), 1e-12)
}

func (s *testSuite) TestLoadSkipsTilesPastTheEndOfTheMap() {
	persistence := newFakePersistence()
	persistence.rows = map[uint32]string{10: "fr", maxIndex + 1: "us"}

	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Load(context.Background()))

	state, err := stateBatch(storage, 0, maxIndex)
	s.Require().NoError(err)
	s.Equal(map[uint32]string{10: "fr"}, state)
}

func (s *testSuite) TestAFailedLoadRefusesTheBoot() {
	persistence := newFakePersistence()
	persistence.fail(errors.New("connection refused"))

	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().ErrorContains(storage.Load(context.Background()), "connection refused")
}

func (s *testSuite) TestFlushWritesEachChangedTileOnceAsItIsNow() {
	ctx := context.Background()
	persistence := newFakePersistence()
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().NoError(storage.Set(ctx, 1, "fr"))
	s.Require().NoError(storage.Set(ctx, 1, "us"))
	s.Require().NoError(storage.Set(ctx, 70, "fr"))
	s.Require().NoError(storage.Set(ctx, 2, "fr"))
	_, err := storage.Clear(ctx, clicks.Blast{Cleared: []uint32{2}})
	s.Require().NoError(err)

	s.Require().NoError(storage.Flush(ctx))

	s.Equal([]save{{tiles: []uint32{1, 2, 70}, owners: []string{"us", "", "fr"}}}, persistence.saves)
	s.Equal(map[uint32]string{1: "us", 70: "fr"}, persistence.stored())
}

func (s *testSuite) TestFlushWithNothingChangedWritesNothing() {
	ctx := context.Background()
	persistence := newFakePersistence()
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().NoError(storage.Set(ctx, 1, "fr"))
	s.Require().NoError(storage.Flush(ctx))
	s.Require().NoError(storage.Flush(ctx))
	s.Require().NoError(storage.Set(ctx, 1, "fr"))
	s.Require().NoError(storage.Flush(ctx))

	s.Len(persistence.saves, 1, "a click on a tile already held changes nothing to write")
}

func (s *testSuite) TestAFailedFlushKeepsTheTilesForTheNextOne() {
	ctx := context.Background()
	persistence := newFakePersistence()
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().NoError(storage.Set(ctx, 1, "fr"))
	persistence.fail(errors.New("connection reset"))
	s.Require().Error(storage.Flush(ctx))

	persistence.fail(nil)
	s.Require().NoError(storage.Set(ctx, 2, "us"))
	s.Require().NoError(storage.Flush(ctx))

	s.Equal(map[uint32]string{1: "fr", 2: "us"}, persistence.stored())
}

func (s *testSuite) TestRestoredTilesAreFlushed() {
	ctx := context.Background()
	persistence := newFakePersistence()
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().NoError(storage.Set(ctx, 3, "ps"))
	s.Require().NoError(storage.Flush(ctx))
	_, err := storage.Restore(ctx, []clicks.Restoration{{Tile: 3, From: "ps", To: ""}})
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))

	s.Empty(persistence.stored())
}

func (s *testSuite) TestRunFlushesOnShutdown() {
	persistence := newFakePersistence()
	storage := s.newStorageOn(inmemory_tile_storage.Config{FlushInterval: time.Hour}, persistence)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		storage.Run(ctx)
	}()

	s.Require().NoError(storage.Set(context.Background(), 7, "fr"))
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		s.T().Fatal("Run did not return after the context was cancelled")
	}

	s.Equal(map[uint32]string{7: "fr"}, persistence.stored())
}

func (s *testSuite) TestRunFlushesOnItsInterval() {
	persistence := newFakePersistence()
	storage := s.newStorageOn(inmemory_tile_storage.Config{FlushInterval: 10 * time.Millisecond}, persistence)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go storage.Run(ctx)

	s.Require().NoError(storage.Set(context.Background(), 7, "fr"))

	s.Eventually(func() bool {
		return persistence.stored()[7] == "fr"
	}, 5*time.Second, 10*time.Millisecond)
}

func (s *testSuite) TestReassignedTilesSurviveAFlush() {
	ctx := context.Background()
	persistence := newFakePersistence()

	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Set(ctx, 12, "dz"))
	_, _, err := storage.Reassign(ctx, "dz", "fr", 0, 10)
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))

	restored := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(restored.Load(ctx))
	owner, _ := restored.Owner(12)
	s.Equal("fr", owner)
	s.Equal(1, restored.Held("fr"))
	s.Zero(restored.Held("dz"))
}
