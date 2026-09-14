package memory_tile_storage_test

import (
	"context"
	"encoding/binary"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/stretchr/testify/suite"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	storage *memory_tile_storage.Storage
}

const maxIndex = 100_000

func (s *testSuite) SetupTest() {
	s.storage = s.newStorage(memory_tile_storage.Config{})
}

func (s *testSuite) newStorage(cfg memory_tile_storage.Config) *memory_tile_storage.Storage {
	return s.newStorageOn(cfg, newFakePersistence())
}

func (s *testSuite) newStorageOn(cfg memory_tile_storage.Config, persistence memory_tile_storage.Persistence) *memory_tile_storage.Storage {
	return memory_tile_storage.New(maxIndex, cfg, persistence, slog.New(slog.DiscardHandler))
}

func (s *testSuite) TestSetAndPublish() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	listener, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	s.Require().NoError(s.storage.Set(context.Background(), 10, "fr"))

	select {
	case <-ctx.Done():
		s.T().Fatal("timeout")
	case val := <-listener:
		s.Require().NotNil(val.Update)
		s.Equal("fr", val.Update.Value)
		s.Equal(uint32(10), val.Update.Tile)
		s.Empty(val.Update.Previous)
	}
}

func (s *testSuite) TestSetBoostedMarksTheUpdate() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	listener, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	s.Require().NoError(s.storage.Set(context.Background(), 11, "fr"))
	s.Require().NoError(s.storage.SetBoosted(context.Background(), 12, "fr"))

	for _, boosted := range []bool{false, true} {
		select {
		case <-ctx.Done():
			s.T().Fatal("timeout")
		case val := <-listener:
			s.Require().NotNil(val.Update)
			s.Equal(boosted, val.Update.Boosted)
		}
	}
}

func (s *testSuite) TestSetAndPublishWithOverride() {
	previousValue, newValue := "us", "fr"

	s.Require().NoError(s.storage.Set(context.Background(), 10, previousValue))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	listener, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	s.Require().NoError(s.storage.Set(context.Background(), 10, newValue))

	select {
	case <-ctx.Done():
		s.T().Fatal("timeout")
	case val := <-listener:
		s.Require().NotNil(val.Update)
		s.Equal(newValue, val.Update.Value)
		s.Equal(uint32(10), val.Update.Tile)
		s.Equal(previousValue, val.Update.Previous)
	}
}

func (s *testSuite) TestSetAndPublishWithOverrideAndNoChange() {
	constantValue := "fr"

	s.Require().NoError(s.storage.Set(context.Background(), 10, constantValue))

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	listener, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	s.Require().NoError(s.storage.Set(context.Background(), 10, constantValue))

	select {
	case <-ctx.Done():
		s.T().Logf("as expected, no message was received")
	case val, ok := <-listener:
		if ok {
			s.T().Errorf("unexpected value %v", val)
		}
	}
}

func (s *testSuite) TestSetOutOfRange() {
	s.Error(s.storage.Set(context.Background(), maxIndex+1, "fr"))
}

func (s *testSuite) TestSubscribeFansOutToEverySubscriber() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	const subscribers = 5
	listeners := make([]<-chan clicks.Change, 0, subscribers)
	for range subscribers {
		listener, err := s.storage.Subscribe(ctx)
		s.Require().NoError(err)
		listeners = append(listeners, listener)
	}

	s.Require().NoError(s.storage.Set(context.Background(), 42, "fr"))

	for i, listener := range listeners {
		select {
		case <-ctx.Done():
			s.T().Fatalf("subscriber %d timed out", i)
		case val := <-listener:
			s.Require().NotNil(val.Update)
			s.Equal(uint32(42), val.Update.Tile)
			s.Equal("fr", val.Update.Value)
		}
	}
}

func (s *testSuite) TestClearEmptiesTheTilesAndPublishesOneBlast() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	s.Require().NoError(s.storage.Set(ctx, 10, "fr"))
	s.Require().NoError(s.storage.Set(ctx, 12, "jp"))

	listener, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	blast, err := s.storage.Clear(ctx, clicks.Blast{Tile: 11, CountryID: "de", Cleared: []uint32{10, 11, 12}})
	s.Require().NoError(err)
	s.Equal([]uint32{10, 12}, blast.Cleared, "only tiles that were held are reported")

	select {
	case <-ctx.Done():
		s.T().Fatal("timeout")
	case val := <-listener:
		s.Require().NotNil(val.Blast)
		s.Equal([]uint32{10, 12}, val.Blast.Cleared)
		s.Equal("de", val.Blast.CountryID)
	}
	s.Empty(listener, "one event for the blast, not one per tile")

	owner, _ := s.storage.Owner(10)
	s.Empty(owner)
}

func (s *testSuite) TestClearRefusesATileOutOfRange() {
	_, err := s.storage.Clear(context.Background(), clicks.Blast{Cleared: []uint32{maxIndex + 1}})
	s.Error(err)
}

func (s *testSuite) TestSubscribeClosesChannelOnContextCancel() {
	ctx, cancel := context.WithCancel(context.Background())

	listener, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	cancel()

	select {
	case _, ok := <-listener:
		s.False(ok, "channel should be closed")
	case <-time.After(2 * time.Second):
		s.T().Fatal("channel was not closed after the context was cancelled")
	}

	s.NoError(s.storage.Set(context.Background(), 1, "fr"))
}

func (s *testSuite) TestSlowSubscriberIsDroppedNotBlocking() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storage := s.newStorage(memory_tile_storage.Config{SubscriberBuffer: 1})

	listener, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	errs := make(chan error, 1)
	go func() {
		defer close(errs)
		for i := uint32(1); i <= 1000; i++ {
			if err := storage.Set(context.Background(), i, "fr"); err != nil {
				errs <- err
				return
			}
		}
	}()

	select {
	case err := <-errs:
		s.Require().NoError(err)
	case <-time.After(5 * time.Second):
		s.T().Fatal("a slow subscriber blocked the writers")
	}

	s.Equal(uint64(999), storage.DroppedUpdates())
	s.Len(listener, 1)

	state, err := stateBatch(storage, 1, 1000)
	s.Require().NoError(err)
	s.Len(state, 1000)
}

func (s *testSuite) TestGetStateByBatch() {
	constantValue := "fr"

	for _, tile := range []uint32{10, 20, 30} {
		s.Require().NoError(s.storage.Set(context.Background(), tile, constantValue))
	}

	state, err := stateBatch(s.storage, 10, 30)
	s.Require().NoError(err)

	s.Len(state, 3)
	s.Equal(constantValue, state[10])
	s.Equal(constantValue, state[20])
	s.Equal(constantValue, state[30])
}

func (s *testSuite) TestGetStateByBatchIgnoresUnsetAndOutOfRangeTiles() {
	s.Require().NoError(s.storage.Set(context.Background(), 10, "fr"))

	state, err := stateBatch(s.storage, 5, maxIndex+1_000)
	s.Require().NoError(err)
	s.Equal(map[uint32]string{10: "fr"}, state)
}

func (s *testSuite) TestShareFollowsSetsAndBlasts() {
	ctx := context.Background()
	for tile := uint32(1); tile <= 10; tile++ {
		s.Require().NoError(s.storage.Set(ctx, tile, "bg"))
	}
	s.Require().NoError(s.storage.Set(ctx, 3, "fr"))
	s.Require().NoError(s.storage.Set(ctx, 4, "bg"))

	s.InDelta(9.0/maxIndex, s.storage.Share("bg"), 1e-12)
	s.InDelta(1.0/maxIndex, s.storage.Share("fr"), 1e-12)

	_, err := s.storage.Clear(ctx, clicks.Blast{Cleared: []uint32{1, 2, 3, 50}})
	s.Require().NoError(err)

	s.InDelta(7.0/maxIndex, s.storage.Share("bg"), 1e-12)
	s.Zero(s.storage.Share("fr"))
	s.Zero(s.storage.Share("de"), "a country that never clicked holds nothing")
	s.Zero(s.storage.Share(""), "unowned ground is nobody's share")
}

func (s *testSuite) TestConcurrentSetsAndReads() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const (
		writers        = 8
		tilesPerWriter = 2_000
	)

	storage := s.newStorage(memory_tile_storage.Config{FlushInterval: time.Millisecond})

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

var _ click.TileStorage = (*memory_tile_storage.Storage)(nil)

func stateBatch(s *memory_tile_storage.Storage, start uint32, end uint32) (map[uint32]string, error) {
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
