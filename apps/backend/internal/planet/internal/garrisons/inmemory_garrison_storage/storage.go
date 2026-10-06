package inmemory_garrison_storage

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Persistence interface {
	Load(ctx context.Context, visit func(garrison garrisons.Garrison)) error
	Save(ctx context.Context, changed []garrisons.Garrison) error
}

func New(config Config, perTile int, persistence Persistence, logger *slog.Logger) *Storage {
	return &Storage{
		config:      config.withDefaults(),
		perTile:     perTile,
		persistence: persistence,
		logger:      logger,
		garrisons:   make(map[uint32]garrisons.Garrison),
		dirty:       cpcolls.NewSet[uint32](),
		subscribers: cpcolls.NewSet[chan garrisons.Garrison](),
	}
}

type Storage struct {
	config      Config
	perTile     int
	persistence Persistence
	logger      *slog.Logger

	mu        sync.Mutex
	garrisons map[uint32]garrisons.Garrison
	dirty     *cpcolls.Set[uint32]

	subscribersMu sync.Mutex
	subscribers   *cpcolls.Set[chan garrisons.Garrison]

	flushMu sync.Mutex
}

func (s *Storage) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.persistence.Load(ctx, func(garrison garrisons.Garrison) {
		s.garrisons[garrison.Tile] = garrison
	})
	if err != nil {
		return fmt.Errorf("failed to read the stored garrisons: %w", err)
	}

	s.logger.Info("loaded the garrisons", slog.Int("tiles", len(s.garrisons)))

	return nil
}

func (s *Storage) PerTile() int {
	return s.perTile
}

func (s *Storage) Defenders(tile uint32, owner string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.garrisons[tile].Standing(owner)
}

func (s *Storage) Full(tile uint32, country string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.garrisons[tile].Full(country, s.perTile)
}

func (s *Storage) Garrisons() []garrisons.Garrison {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.SortedFunc(maps.Values(s.garrisons), func(a, b garrisons.Garrison) int {
		return cmp.Compare(a.Tile, b.Tile)
	})
}

func (s *Storage) Reinforce(tile uint32, country string) {
	s.mu.Lock()
	garrison := s.garrisons[tile]
	garrison.Tile = tile
	reinforced := garrison.Reinforced(country, s.perTile)
	s.putLocked(reinforced)
	s.mu.Unlock()

	if reinforced != garrison {
		s.publish(reinforced)
	}
}

func (s *Storage) Strike(tile uint32, owner string) bool {
	s.mu.Lock()
	garrison, ok := s.garrisons[tile]
	if !ok {
		s.mu.Unlock()
		return false
	}
	struck, hit := garrison.Struck(owner)
	s.putLocked(struck)
	s.mu.Unlock()

	if hit {
		s.publish(struck)
	}

	return hit
}

func (s *Storage) putLocked(garrison garrisons.Garrison) {
	if garrison.Empty() {
		delete(s.garrisons, garrison.Tile)
	} else {
		s.garrisons[garrison.Tile] = garrison
	}
	s.dirty.Add(garrison.Tile)
}

func (s *Storage) Subscribe(ctx context.Context) (<-chan garrisons.Garrison, error) {
	updates := make(chan garrisons.Garrison, s.config.SubscriberBuffer)

	s.subscribersMu.Lock()
	s.subscribers.Add(updates)
	s.subscribersMu.Unlock()

	go func() {
		<-ctx.Done()

		s.subscribersMu.Lock()
		defer s.subscribersMu.Unlock()

		s.subscribers.Delete(updates)
		close(updates)
	}()

	return updates, nil
}

func (s *Storage) publish(garrison garrisons.Garrison) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	s.subscribers.ForEach(func(updates chan garrisons.Garrison) {
		select {
		case updates <- garrison:
		default:
		}
	})
}

func (s *Storage) Name() string { return "garrison-storage" }

func (s *Storage) Run(ctx context.Context) {
	ticker := time.NewTicker(s.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.flushOrLog(ctx)
		case <-ctx.Done():
			s.flushOrLog(context.WithoutCancel(ctx))
			return
		}
	}
}

func (s *Storage) flushOrLog(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, flushTimeout)
	defer cancel()

	if err := s.Flush(ctx); err != nil {
		s.logger.Error("failed to flush the garrisons, retrying next tick", slog.Any("error", err))
	}
}

func (s *Storage) Flush(ctx context.Context) error {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	s.mu.Lock()
	if s.dirty.Empty() {
		s.mu.Unlock()
		return nil
	}

	changed := make([]garrisons.Garrison, 0, s.dirty.Len())
	s.dirty.ForEach(func(tile uint32) {
		garrison := s.garrisons[tile]
		garrison.Tile = tile
		changed = append(changed, garrison)
	})
	s.dirty = cpcolls.NewSet[uint32]()
	s.mu.Unlock()

	if err := s.persistence.Save(ctx, changed); err != nil {
		s.mu.Lock()
		for _, garrison := range changed {
			s.dirty.Add(garrison.Tile)
		}
		s.mu.Unlock()

		return fmt.Errorf("failed to save %d garrisons: %w", len(changed), err)
	}

	return nil
}
