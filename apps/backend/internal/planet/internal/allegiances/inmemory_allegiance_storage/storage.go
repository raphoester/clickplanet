package inmemory_allegiance_storage

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Persistence interface {
	Load(ctx context.Context, visit func(account allegiances.Account, kept allegiances.Snapshot)) error
	Save(ctx context.Context, kept map[allegiances.Account]allegiances.Snapshot) error
}

type Config struct {
	FlushInterval time.Duration
}

const (
	defaultFlushInterval = time.Second
	flushTimeout         = 10 * time.Second
)

func (c Config) withDefaults() Config {
	if c.FlushInterval <= 0 {
		c.FlushInterval = defaultFlushInterval
	}

	return c
}

func New(config Config, persistence Persistence, logger *slog.Logger) *Storage {
	return &Storage{
		config:      config.withDefaults(),
		persistence: persistence,
		logger:      logger,
		kept:        make(map[allegiances.Account]allegiances.Allegiance),
		dirty:       cpcolls.NewSet[allegiances.Account](),
	}
}

type Storage struct {
	config      Config
	persistence Persistence
	logger      *slog.Logger

	mu    sync.Mutex
	kept  map[allegiances.Account]allegiances.Allegiance
	dirty *cpcolls.Set[allegiances.Account]

	flushMu sync.Mutex
}

var _ allegiances.Store = (*Storage)(nil)

func (s *Storage) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.persistence.Load(ctx, func(account allegiances.Account, kept allegiances.Snapshot) {
		s.kept[account] = kept.Restore()
	})
	if err != nil {
		return fmt.Errorf("failed to read the stored allegiances: %w", err)
	}

	s.logger.Info("loaded the allegiances", slog.Int("accounts", len(s.kept)))

	return nil
}

func (s *Storage) Change(account allegiances.Account, change func(allegiances.Allegiance) (allegiances.Allegiance, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := s.kept[account]
	after, err := change(before)
	if err != nil {
		return err
	}

	if after.TakeSnapshot() != before.TakeSnapshot() {
		s.kept[account] = after
		s.dirty.Add(account)
	}

	return nil
}

func (s *Storage) Snapshot(account allegiances.Account) allegiances.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.kept[account].TakeSnapshot()
}

func (s *Storage) Name() string { return "allegiance-storage" }

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
		s.logger.Error("failed to flush the allegiances, retrying next tick", slog.Any("error", err))
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

	changed := make(map[allegiances.Account]allegiances.Snapshot, s.dirty.Len())
	s.dirty.ForEach(func(account allegiances.Account) { changed[account] = s.kept[account].TakeSnapshot() })
	s.dirty = cpcolls.NewSet[allegiances.Account]()
	s.mu.Unlock()

	if err := s.persistence.Save(ctx, changed); err != nil {
		s.mu.Lock()
		for account := range changed {
			s.dirty.Add(account)
		}
		s.mu.Unlock()

		return fmt.Errorf("failed to save %d allegiances: %w", len(changed), err)
	}

	return nil
}
