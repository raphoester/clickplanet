package inmemory_tile_storage_test

import (
	"context"
	"encoding/binary"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/stretchr/testify/suite"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	clicks.TileStorageContractSuite
}

const maxIndex = 100_000

var island = []uint32{90_001, 90_002, 90_003}

var borders = clicks.BordersOf(maxIndex, island)

func (s *testSuite) SetupSuite() {
	s.NewStorage = func(borders *clicks.Borders) clicks.TileStorage {
		return inmemory_tile_storage.New(borders, inmemory_tile_storage.Config{}, inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}), slog.New(slog.DiscardHandler))
	}
}

func (s *testSuite) newStorage(cfg inmemory_tile_storage.Config) *inmemory_tile_storage.Storage {
	return s.newStorageOn(cfg, inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
}

func (s *testSuite) newStorageOn(cfg inmemory_tile_storage.Config, persistence inmemory_tile_storage.Persistence) *inmemory_tile_storage.Storage {
	return inmemory_tile_storage.New(borders, cfg, persistence, slog.New(slog.DiscardHandler))
}

func (s *testSuite) TestASlowSubscriberIsClosedRatherThanHoldingTheWriters() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storage := s.newStorage(inmemory_tile_storage.Config{SubscriberBuffer: 1})

	slow, err := storage.Subscribe(ctx)
	s.Require().NoError(err)
	reading, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	for tile := uint32(1); tile <= 1000; tile++ {
		s.Require().NoError(storage.Set(context.Background(), tile, "fr"))

		change, open := <-reading
		s.Require().True(open, "a subscriber that keeps up is never cut off")
		s.Require().Equal(tile, change.Update.Tile)
	}

	change, open := <-slow
	s.Require().True(open)
	s.Equal(uint32(1), change.Update.Tile)
	_, open = <-slow
	s.False(open, "closed, not skipped: its stream ends and the client reads the map again")

	state, err := stateBatch(storage, 1, 1000)
	s.Require().NoError(err)
	s.Len(state, 1000)
}

func (s *testSuite) TestConcurrentSetsAndReads() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const (
		writers        = 8
		tilesPerWriter = 2_000
	)

	storage := s.newStorage(inmemory_tile_storage.Config{FlushInterval: time.Millisecond})

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		storage.Run(ctx)
	}()
	defer func() {
		cancel()
		select {
		case <-stopped:
		case <-time.After(5 * time.Second):
			s.T().Error("Run did not return after the context was cancelled")
		}
	}()

	received := make(chan struct{})
	listener, err := storage.Subscribe(ctx)
	s.Require().NoError(err)
	go func() {
		defer close(received)
		for range listener {
		}
	}()

	_, err = storage.Subscribe(ctx)
	s.Require().NoError(err)

	countries := []string{"fr", "us", "de", "es"}

	errs := make(chan error, 2*writers)

	wg := sync.WaitGroup{}
	for w := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range tilesPerWriter {
				tile := uint32(w*tilesPerWriter + i + 1)
				if err := storage.Set(context.Background(), tile, countries[i%len(countries)]); err != nil {
					errs <- err
					return
				}
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				if _, err := stateBatch(storage, 1, writers*tilesPerWriter); err != nil {
					errs <- err
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		s.Require().NoError(err)
	}

	state, err := stateBatch(storage, 1, writers*tilesPerWriter)
	s.Require().NoError(err)
	s.Len(state, writers*tilesPerWriter)

	cancel()
	select {
	case <-received:
	case <-time.After(5 * time.Second):
		s.T().Fatal("the subscriber channel was never closed")
	}
}

func (s *testSuite) TestTheTerritoriesAreTheTilesEachCountryHoldsAndNoneForACountryThatHoldsNone() {
	storage := s.newStorage(inmemory_tile_storage.Config{})
	ctx := context.Background()
	s.Require().NoError(storage.Set(ctx, 1, "fr"))
	s.Require().NoError(storage.Set(ctx, 2, "fr"))
	s.Require().NoError(storage.Set(ctx, 3, "de"))
	s.Require().NoError(storage.Set(ctx, 4, "it"))

	s.Require().NoError(storage.Set(ctx, 4, "fr"))
	_, err := storage.Clear(ctx, clicks.Blast{Cleared: []uint32{3}})
	s.Require().NoError(err)

	s.Equal(map[string]uint32{"fr": 3}, storage.Territories())
}

var _ clicks.Claimable = (*inmemory_tile_storage.Storage)(nil)

func stateBatch(s *inmemory_tile_storage.Storage, start uint32, end uint32) (map[uint32]string, error) {
	batch, err := s.StateBatchDense(start, end)
	if err != nil {
		return nil, err
	}

	state := make(map[uint32]string)
	for i := 0; i+1 < len(batch.Tiles); i += 2 {
		code := binary.LittleEndian.Uint16(batch.Tiles[i : i+2])
		if code == 0 {
			continue
		}
		state[batch.Start+uint32(i/2)] = batch.Codes[code]
	}

	return state, nil
}
