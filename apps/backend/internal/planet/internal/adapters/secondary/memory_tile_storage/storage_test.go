package memory_tile_storage_test

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/logging"
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
	return memory_tile_storage.New(maxIndex, cfg, logging.NewNopLogger())
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
		s.Assert().Equal("fr", val.Value)
		s.Assert().Equal(uint32(10), val.Tile)
		s.Assert().Equal("", val.Previous)
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
		s.Assert().Equal(newValue, val.Value)
		s.Assert().Equal(uint32(10), val.Tile)
		s.Assert().Equal(previousValue, val.Previous)
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
	s.Assert().Error(s.storage.Set(context.Background(), maxIndex+1, "fr"))
}

func (s *testSuite) TestSubscribeFansOutToEverySubscriber() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	const subscribers = 5
	listeners := make([]<-chan domain.TileUpdate, 0, subscribers)
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
			s.Assert().Equal(uint32(42), val.Tile)
			s.Assert().Equal("fr", val.Value)
		}
	}
}

func (s *testSuite) TestSubscribeClosesChannelOnContextCancel() {
	ctx, cancel := context.WithCancel(context.Background())

	listener, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	cancel()

	select {
	case _, ok := <-listener:
		s.Assert().False(ok, "channel should be closed")
	case <-time.After(2 * time.Second):
		s.T().Fatal("channel was not closed after the context was cancelled")
	}

	s.Assert().NoError(s.storage.Set(context.Background(), 1, "fr"))
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

	s.Assert().Equal(uint64(999), storage.DroppedUpdates())
	s.Assert().Len(listener, 1)

	state, err := stateBatch(storage, 1, 1000)
	s.Require().NoError(err)
	s.Assert().Len(state, 1000)
}

func (s *testSuite) TestGetStateByBatch() {
	constantValue := "fr"

	for _, tile := range []uint32{10, 20, 30} {
		s.Require().NoError(s.storage.Set(context.Background(), tile, constantValue))
	}

	state, err := stateBatch(s.storage, 10, 30)
	s.Require().NoError(err)

	s.Assert().Equal(3, len(state))
	s.Assert().Equal(constantValue, state[10])
	s.Assert().Equal(constantValue, state[20])
	s.Assert().Equal(constantValue, state[30])
}

func (s *testSuite) TestGetStateByBatchIgnoresUnsetAndOutOfRangeTiles() {
	s.Require().NoError(s.storage.Set(context.Background(), 10, "fr"))

	state, err := stateBatch(s.storage, 5, maxIndex+1_000)
	s.Require().NoError(err)
	s.Assert().Equal(map[uint32]string{10: "fr"}, state)
}

func (s *testSuite) TestSnapshotRoundTrip() {
	path := filepath.Join(s.T().TempDir(), "tiles.snapshot")
	cfg := memory_tile_storage.Config{SnapshotPath: path}

	storage := s.newStorage(cfg)
	s.Require().NoError(storage.Set(context.Background(), 1, "fr"))
	s.Require().NoError(storage.Set(context.Background(), 2, "us"))
	s.Require().NoError(storage.Set(context.Background(), maxIndex, "de"))
	s.Require().NoError(storage.Snapshot())

	restored := s.newStorage(cfg)
	state, err := stateBatch(restored, 0, maxIndex)
	s.Require().NoError(err)

	s.Assert().Equal(map[uint32]string{1: "fr", 2: "us", maxIndex: "de"}, state)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	listener, err := restored.Subscribe(ctx)
	s.Require().NoError(err)
	s.Require().NoError(restored.Set(context.Background(), 1, "us"))

	select {
	case <-ctx.Done():
		s.T().Fatal("timeout")
	case update := <-listener:
		s.Assert().Equal("fr", update.Previous)
		s.Assert().Equal("us", update.Value)
	}
}

func (s *testSuite) TestSnapshotIsSkippedWhenNothingChanged() {
	path := filepath.Join(s.T().TempDir(), "tiles.snapshot")
	cfg := memory_tile_storage.Config{SnapshotPath: path}

	storage := s.newStorage(cfg)
	s.Require().NoError(storage.Set(context.Background(), 1, "fr"))
	s.Require().NoError(storage.Snapshot())

	before, err := os.Stat(path)
	s.Require().NoError(err)

	s.Require().NoError(storage.Snapshot())

	after, err := os.Stat(path)
	s.Require().NoError(err)
	s.Assert().Equal(before.ModTime(), after.ModTime())
}

func (s *testSuite) TestSnapshotWritesNoTempFileBehind() {
	dir := s.T().TempDir()
	path := filepath.Join(dir, "tiles.snapshot")

	storage := s.newStorage(memory_tile_storage.Config{SnapshotPath: path})
	s.Require().NoError(storage.Set(context.Background(), 1, "fr"))
	s.Require().NoError(storage.Snapshot())

	entries, err := os.ReadDir(dir)
	s.Require().NoError(err)
	s.Require().Len(entries, 1)
	s.Assert().Equal("tiles.snapshot", entries[0].Name())
}

func (s *testSuite) TestMissingSnapshotStartsEmpty() {
	path := filepath.Join(s.T().TempDir(), "does-not-exist.snapshot")

	storage := s.newStorage(memory_tile_storage.Config{SnapshotPath: path})

	state, err := stateBatch(storage, 0, maxIndex)
	s.Require().NoError(err)
	s.Assert().Empty(state)

	s.Require().NoError(storage.Set(context.Background(), 1, "fr"))
}

func (s *testSuite) TestCorruptSnapshotStartsEmpty() {
	valid := func() []byte {
		path := filepath.Join(s.T().TempDir(), "tiles.snapshot")
		storage := s.newStorage(memory_tile_storage.Config{SnapshotPath: path})
		s.Require().NoError(storage.Set(context.Background(), 1, "fr"))
		s.Require().NoError(storage.Snapshot())
		raw, err := os.ReadFile(path)
		s.Require().NoError(err)
		return raw
	}()

	corruptions := map[string][]byte{
		"empty file":          {},
		"shorter than header": valid[:4],
		"bad magic":           append([]byte("NOTATILE"), valid[8:]...),
		"truncated body":      valid[:len(valid)-10],
		"flipped byte": func() []byte {
			raw := append([]byte(nil), valid...)
			raw[len(raw)-1] ^= 0xff
			return raw
		}(),
		"unknown version": func() []byte {
			raw := append([]byte(nil), valid...)
			raw[8] = 99
			return raw
		}(),
	}

	for name, raw := range corruptions {
		s.Run(name, func() {
			path := filepath.Join(s.T().TempDir(), "tiles.snapshot")
			s.Require().NoError(os.WriteFile(path, raw, 0o644))

			storage := s.newStorage(memory_tile_storage.Config{SnapshotPath: path})

			state, err := stateBatch(storage, 0, maxIndex)
			s.Require().NoError(err)
			s.Assert().Empty(state)

			s.Require().NoError(storage.Set(context.Background(), 1, "fr"))
			s.Require().NoError(storage.Snapshot())
		})
	}
}

func (s *testSuite) TestSnapshotSurvivesADifferentMapSize() {
	path := filepath.Join(s.T().TempDir(), "tiles.snapshot")
	cfg := memory_tile_storage.Config{SnapshotPath: path}

	big := memory_tile_storage.New(1_000, cfg, logging.NewNopLogger())
	s.Require().NoError(big.Set(context.Background(), 10, "fr"))
	s.Require().NoError(big.Set(context.Background(), 900, "us"))
	s.Require().NoError(big.Snapshot())

	small := memory_tile_storage.New(100, cfg, logging.NewNopLogger())
	state, err := stateBatch(small, 0, 100)
	s.Require().NoError(err)
	s.Assert().Equal(map[uint32]string{10: "fr"}, state)
}

func (s *testSuite) TestRunSnapshotsPeriodicallyAndOnShutdown() {
	path := filepath.Join(s.T().TempDir(), "tiles.snapshot")
	cfg := memory_tile_storage.Config{SnapshotPath: path, SnapshotInterval: time.Hour}

	storage := s.newStorage(cfg)

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

	restored := s.newStorage(cfg)
	state, err := stateBatch(restored, 0, maxIndex)
	s.Require().NoError(err)
	s.Assert().Equal(map[uint32]string{7: "fr"}, state)
}

func (s *testSuite) TestRunWithoutSnapshotPathReturnsOnCancel() {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.storage.Run(ctx)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		s.T().Fatal("Run did not return after the context was cancelled")
	}
}

func (s *testSuite) TestConcurrentSetsAndReads() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const (
		writers        = 8
		tilesPerWriter = 2_000
	)

	storage := s.newStorage(memory_tile_storage.Config{
		SnapshotPath:     filepath.Join(s.T().TempDir(), "tiles.snapshot"),
		SnapshotInterval: time.Millisecond,
	})

	// Joined before returning: a snapshot landing mid-cleanup breaks RemoveAll.
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
	s.Assert().Len(state, writers*tilesPerWriter)

	cancel()
	select {
	case <-received:
	case <-time.After(5 * time.Second):
		s.T().Fatal("the subscriber channel was never closed")
	}
}

var _ domain.TileStorage = (*memory_tile_storage.Storage)(nil)

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
