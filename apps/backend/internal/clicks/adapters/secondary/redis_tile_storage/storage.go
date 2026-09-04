package redis_tile_storage

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	SetAndPublishOnStreamSha1 string
}

func New(
	redis *redis.Client,
	config Config,
	logger logging.Logger,
) *Storage {
	return &Storage{
		redis:                     redis,
		setAndPublishOnStreamSha1: config.SetAndPublishOnStreamSha1,
		logger:                    logger,
	}
}

const streamLabel = "tileUpdates"

type Storage struct {
	redis                     *redis.Client
	setAndPublishOnStreamSha1 string
	logger                    logging.Logger
}

func (s *Storage) Set(ctx context.Context, tile uint32, value string) error {
	_, err := s.redis.EvalSha(
		ctx,
		s.setAndPublishOnStreamSha1,
		[]string{strconv.FormatUint(uint64(tile), 10)},
		value, streamLabel,
	).Result()

	if err != nil {
		return fmt.Errorf("failed to set tile: %w", err)
	}

	return nil
}

func (s *Storage) GetStateBatch(ctx context.Context, start uint32, end uint32) (map[uint32]string, error) {
	keys := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		keys = append(keys, strconv.FormatUint(uint64(i), 10))
	}

	values, err := s.redis.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get tile values: %w", err)
	}

	retMap := make(map[uint32]string, len(values))
	for i, key := range keys {
		if values[i] == nil {
			continue
		}

		tile, err := strconv.ParseUint(key, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse tileId to int: %w", err)
		}

		retMap[uint32(tile)] = values[i].(string)
	}

	return retMap, nil
}

// Subscribe streams every tile update published after the call returns.
//
// The read position is resolved to a concrete stream ID *before* returning, so
// callers can rely on "Subscribe returned" meaning "nothing published from now
// on will be missed". Reading with "$" instead would re-resolve the tail on
// every XREAD, silently dropping everything appended between two reads.
func (s *Storage) Subscribe(ctx context.Context) (<-chan domain.TileUpdate, error) {
	lastID, err := s.streamTailID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to pin stream position: %w", err)
	}

	ch := make(chan domain.TileUpdate)

	go func() {
		defer close(ch)

		for {
			if ctx.Err() != nil {
				return
			}

			streams, err := s.redis.XRead(ctx, &redis.XReadArgs{
				Streams: []string{streamLabel, lastID},
				Count:   100,
				Block:   1 * time.Second, // DO NOT SET AT 0 OTHERWISE IT WILL BLOCK FOREVER
			}).Result()

			if err != nil {
				if errors.Is(err, redis.Nil) { // wtf redis lib: no message before the block deadline
					continue
				}

				if ctx.Err() != nil {
					return
				}

				s.logger.Error("failed to read tile updates stream", lf.Err(err))
				continue
			}

			if len(streams) == 0 || len(streams[0].Messages) == 0 {
				continue
			}

			// Advance only once the whole batch has been handed to the consumer:
			// the next XREAD acts as the acknowledgement of the previous one.
			for _, message := range streams[0].Messages {
				tileUpdate, err := xMessageToTileUpdate(message)
				if err != nil {
					s.logger.Error("failed to parse message to tile update",
						lf.String("message_id", message.ID), lf.Err(err))
					lastID = message.ID
					continue
				}

				select {
				case ch <- *tileUpdate:
					lastID = message.ID
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}

// streamTailID returns the ID of the last entry currently in the stream, or
// "0-0" when the stream does not exist yet. Both are safe starting points for
// an XREAD that must only yield entries appended from now on.
func (s *Storage) streamTailID(ctx context.Context) (string, error) {
	messages, err := s.redis.XRevRangeN(ctx, streamLabel, "+", "-", 1).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", fmt.Errorf("failed to read stream tail: %w", err)
	}

	if len(messages) == 0 {
		return "0-0", nil
	}

	return messages[0].ID, nil
}

func (s *Storage) PastUpdates(
	ctx context.Context,
	duration time.Duration,
	now time.Time,
) ([]domain.TileUpdate, error) {
	startTime := now.Add(-duration)
	startID := fmt.Sprintf("%d-0", startTime.UnixMilli())

	response, err := s.redis.XRange(ctx, streamLabel, startID, "+").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to read Redis stream: %w", err)
	}

	updates := make([]domain.TileUpdate, 0, len(response))
	for _, message := range response {
		update, err := xMessageToTileUpdate(message)
		if err != nil {
			return nil, fmt.Errorf("failed to parse message to tile update: %w", err)
		}

		updates = append(updates, *update)
	}

	return updates, nil
}

func xMessageToTileUpdate(message redis.XMessage) (*domain.TileUpdate, error) {
	// reducing bandwidth
	// t : tile
	// n : new value
	// o : old value

	tileId, err := strconv.ParseUint(fmt.Sprint(message.Values["t"]), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tileId to int: %w", err)
	}

	return &domain.TileUpdate{
		Tile:     uint32(tileId),
		Value:    message.Values["n"].(string),
		Previous: message.Values["o"].(string),
	}, nil
}
