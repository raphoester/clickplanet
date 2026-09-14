package inmemory_tile_storage_test

import (
	"context"
	"encoding/binary"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/stretchr/testify/suite"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	clicks.TileStorageContractSuite
}

const maxIndex = 100_000

func (s *testSuite) SetupSuite() {
	s.NewStorage = func(maxIndex uint32) clicks.TileStorage {
		return inmemory_tile_storage.New(maxIndex, inmemory_tile_storage.Config{}, slog.New(slog.DiscardHandler))
	}
}

func (s *testSuite) newStorage(cfg inmemory_tile_storage.Config) *inmemory_tile_storage.Storage {
	storage := inmemory_tile_storage.New(maxIndex, cfg, slog.New(slog.DiscardHandler))
	storage.LoadSnapshot()
	return storage
}

func (s *testSuite) TestSlowSubscriberIsDroppedNotBlocking() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storage := s.newStorage(inmemory_tile_storage.Config{SubscriberBuffer: 1})

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

func (s *testSuite) TestSnapshotRoundTrip() {
	path := filepath.Join(s.T().TempDir(), "tiles.snapshot")
	cfg := inmemory_tile_storage.Config{SnapshotPath: path}

	storage := s.newStorage(cfg)
	s.Require().NoError(storage.Set(context.Background(), 1, "fr"))
	s.Require().NoError(storage.Set(context.Background(), 2, "us"))
	s.Require().NoError(storage.Set(context.Background(), maxIndex, "de"))
	s.Require().NoError(storage.Snapshot())

	restored := s.newStorage(cfg)
	state, err := stateBatch(restored, 0, maxIndex)
	s.Require().NoError(err)

	s.Equal(map[uint32]string{1: "fr", 2: "us", maxIndex: "de"}, state)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	listener, err := restored.Subscribe(ctx)
	s.Require().NoError(err)
	s.Require().NoError(restored.Set(context.Background(), 1, "us"))

	select {
	case <-ctx.Done():
		s.T().Fatal("timeout")
	case change := <-listener:
		s.Require().NotNil(change.Update)
		s.Equal("fr", change.Update.Previous)
		s.Equal("us", change.Update.Value)
	}
}

func (s *testSuite) TestShareIsRebuiltFromTheSnapshot() {
	cfg := inmemory_tile_storage.Config{SnapshotPath: filepath.Join(s.T().TempDir(), "tiles.snapshot")}

	storage := s.newStorage(cfg)
	s.Require().NoError(storage.Set(context.Background(), 1, "bg"))
	s.Require().NoError(storage.Set(context.Background(), 2, "bg"))
	s.Require().NoError(storage.Set(context.Background(), 3, "fr"))
	s.Require().NoError(storage.Snapshot())

	restored := s.newStorage(cfg)
	s.InDelta(2.0/maxIndex, restored.Share("bg"), 1e-12)
	s.InDelta(1.0/maxIndex, restored.Share("fr"), 1e-12)
}

func (s *testSuite) TestSnapshotIsSkippedWhenNothingChanged() {
	path := filepath.Join(s.T().TempDir(), "tiles.snapshot")
	cfg := inmemory_tile_storage.Config{SnapshotPath: path}

	storage := s.newStorage(cfg)
	s.Require().NoError(storage.Set(context.Background(), 1, "fr"))
	s.Require().NoError(storage.Snapshot())

	before, err := os.Stat(path)
	s.Require().NoError(err)

	s.Require().NoError(storage.Snapshot())

	after, err := os.Stat(path)
	s.Require().NoError(err)
	s.Equal(before.ModTime(), after.ModTime())
}

func (s *testSuite) TestSnapshotWritesNoTempFileBehind() {
	dir := s.T().TempDir()
	path := filepath.Join(dir, "tiles.snapshot")

	storage := s.newStorage(inmemory_tile_storage.Config{SnapshotPath: path})
	s.Require().NoError(storage.Set(context.Background(), 1, "fr"))
	s.Require().NoError(storage.Snapshot())

	entries, err := os.ReadDir(dir)
	s.Require().NoError(err)
	s.Require().Len(entries, 1)
	s.Equal("tiles.snapshot", entries[0].Name())
}

func (s *testSuite) TestMissingSnapshotStartsEmpty() {
	path := filepath.Join(s.T().TempDir(), "does-not-exist.snapshot")

	storage := s.newStorage(inmemory_tile_storage.Config{SnapshotPath: path})

	state, err := stateBatch(storage, 0, maxIndex)
	s.Require().NoError(err)
	s.Empty(state)

	s.Require().NoError(storage.Set(context.Background(), 1, "fr"))
}

func (s *testSuite) TestCorruptSnapshotStartsEmpty() {
	valid := func() []byte {
		path := filepath.Join(s.T().TempDir(), "tiles.snapshot")
		storage := s.newStorage(inmemory_tile_storage.Config{SnapshotPath: path})
		s.Require().NoError(storage.Set(context.Background(), 1, "fr"))
		s.Require().NoError(storage.Snapshot())
		//nolint:gosec // G304: path is this test's own t.TempDir() snapshot.
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
			s.Require().NoError(os.WriteFile(path, raw, 0o600))

			storage := s.newStorage(inmemory_tile_storage.Config{SnapshotPath: path})

			state, err := stateBatch(storage, 0, maxIndex)
			s.Require().NoError(err)
			s.Empty(state)

			s.Require().NoError(storage.Set(context.Background(), 1, "fr"))
			s.Require().NoError(storage.Snapshot())
		})
	}
}

func (s *testSuite) TestSnapshotSurvivesADifferentMapSize() {
	path := filepath.Join(s.T().TempDir(), "tiles.snapshot")
	cfg := inmemory_tile_storage.Config{SnapshotPath: path}

	big := inmemory_tile_storage.New(1_000, cfg, slog.New(slog.DiscardHandler))
	s.Require().NoError(big.Set(context.Background(), 10, "fr"))
	s.Require().NoError(big.Set(context.Background(), 900, "us"))
	s.Require().NoError(big.Snapshot())

	small := inmemory_tile_storage.New(100, cfg, slog.New(slog.DiscardHandler))
	small.LoadSnapshot()
	state, err := stateBatch(small, 0, 100)
	s.Require().NoError(err)
	s.Equal(map[uint32]string{10: "fr"}, state)
}

func (s *testSuite) TestRunSnapshotsPeriodicallyAndOnShutdown() {
	path := filepath.Join(s.T().TempDir(), "tiles.snapshot")
	cfg := inmemory_tile_storage.Config{SnapshotPath: path, SnapshotInterval: time.Hour}

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
	s.Equal(map[uint32]string{7: "fr"}, state)
}

func (s *testSuite) TestRunWithoutSnapshotPathReturnsOnCancel() {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.newStorage(inmemory_tile_storage.Config{}).Run(ctx)
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

	storage := s.newStorage(inmemory_tile_storage.Config{
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
	s.Len(state, writers*tilesPerWriter)

	cancel()
	select {
	case <-received:
	case <-time.After(5 * time.Second):
		s.T().Fatal("the subscriber channel was never closed")
	}
}

var _ click_usecase.TileStorage = (*inmemory_tile_storage.Storage)(nil)

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
