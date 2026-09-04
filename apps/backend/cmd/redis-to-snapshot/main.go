// Command redis-to-snapshot copies the tile state out of a running Redis into a
// snapshot file the memory tile storage can boot from.
//
// It is a one-off migration aid for the move off Redis: point it at the same
// config file cmd/api uses, and it writes the file named by
// tilesStorage.memory.snapshotPath.
//
//	go run ./cmd/redis-to-snapshot -config cmd/api/example.yaml
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/redis_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/app"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/cfgutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

// batchSize is how many tiles are pulled per Redis MGET.
const batchSize = 50_000

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "", "path to the cmd/api config file")
	out := flag.String("out", "", "snapshot file to write (defaults to tilesStorage.memory.snapshotPath)")
	force := flag.Bool("force", false, "overwrite the snapshot file if it already exists")
	timeout := flag.Duration("timeout", 5*time.Minute, "overall timeout")
	flag.Parse()

	logger := logging.NewSLogger()

	cfg := app.Config{}
	if err := cfgutil.NewLoader(*configPath).Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed reading config: %w", err)
	}

	snapshotPath := *out
	if snapshotPath == "" {
		snapshotPath = cfg.TilesStorage.Memory.SnapshotPath
	}
	if snapshotPath == "" {
		return errors.New("no snapshot path: set tilesStorage.memory.snapshotPath in the config, or pass -out")
	}

	if err := clearDestination(snapshotPath, *force); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	redisClient, err := xredis.NewClient(ctx, cfg.Redis)
	if err != nil {
		return fmt.Errorf("failed to create redis client: %w", err)
	}
	defer func() { _ = redisClient.Close() }()

	source := redis_tile_storage.New(redisClient, cfg.TilesStorage.Redis)

	// SnapshotPath is set on a destination we just made sure is absent, so
	// nothing is restored and the file below holds exactly what Redis had.
	destination := memory_tile_storage.New(
		cfg.GameMap.MaxIndex,
		memory_tile_storage.Config{SnapshotPath: snapshotPath},
		xtime.ActualProvider{},
		logger,
	)

	copied, err := copyTiles(ctx, source, destination, cfg.GameMap.MaxIndex, logger)
	if err != nil {
		return err
	}

	if err := destination.Snapshot(); err != nil {
		return fmt.Errorf("failed to write snapshot: %w", err)
	}

	logger.Info("wrote snapshot",
		lf.String("path", snapshotPath),
		lf.Int("ownedTiles", copied),
	)

	return nil
}

func clearDestination(path string, force bool) error {
	_, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("failed to stat %q: %w", path, err)
	case !force:
		return fmt.Errorf("%q already exists, pass -force to overwrite it", path)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove %q: %w", path, err)
	}

	return nil
}

func copyTiles(
	ctx context.Context,
	source *redis_tile_storage.Storage,
	destination *memory_tile_storage.Storage,
	maxIndex uint32,
	logger logging.Logger,
) (int, error) {
	copied := 0

	for start := uint32(0); ; start += batchSize {
		end := start + batchSize - 1
		if end > maxIndex {
			end = maxIndex
		}

		batch, err := source.GetStateBatch(ctx, start, end)
		if err != nil {
			return 0, fmt.Errorf("failed to read tiles %d-%d from redis: %w", start, end, err)
		}

		for tile, value := range batch {
			if err := destination.Set(ctx, tile, value); err != nil {
				return 0, fmt.Errorf("failed to set tile %d: %w", tile, err)
			}
			copied++
		}

		logger.Debug("copied a batch",
			lf.Any("start", start),
			lf.Any("end", end),
			lf.Int("ownedTilesSoFar", copied),
		)

		if end == maxIndex {
			return copied, nil
		}
	}
}
