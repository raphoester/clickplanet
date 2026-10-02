package inmemory_charge_storage

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Persistence interface {
	Load(ctx context.Context, visit func(holder bonuses.Holder, held bonuses.Held)) error
	Save(ctx context.Context, hands map[bonuses.Holder]bonuses.Held) error
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

func New(
	config Config,
	rules bonuses.ChargesConfig,
	persistence Persistence,
	logger *slog.Logger,
) *Storage {
	return &Storage{
		config:      config.withDefaults(),
		rules:       rules,
		persistence: persistence,
		logger:      logger,
		hands:       make(map[bonuses.Holder]bonuses.Held),
		dirty:       cpcolls.NewSet[bonuses.Holder](),
	}
}

type Storage struct {
	config      Config
	rules       bonuses.ChargesConfig
	persistence Persistence
	logger      *slog.Logger

	mu    sync.Mutex
	hands map[bonuses.Holder]bonuses.Held
	dirty *cpcolls.Set[bonuses.Holder]

	flushMu sync.Mutex
}

func (s *Storage) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.persistence.Load(ctx, func(holder bonuses.Holder, hand bonuses.Held) {
		if hand.Empty() {
			s.dirty.Add(holder)
			return
		}
		s.hands[holder] = hand
	})
	if err != nil {
		return fmt.Errorf("failed to read the stored charges: %w", err)
	}

	s.logger.Info("loaded the charges", slog.Int("holders", len(s.hands)))

	return nil
}

func (s *Storage) EnclosureMaxTiles() int {
	return s.rules.EnclosureMaxTiles
}

func (s *Storage) SpreadClicks() int {
	return s.rules.SpreadClicks
}

func (s *Storage) Enclosures() int {
	return s.rules.Enclosures
}

func (s *Storage) Grant(holder bonuses.Holder, kind bonuses.Kind, amount int) {
	if holder == bonuses.NoHolder {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.putLocked(holder, s.hands[holder].Granted(kind, amount, s.rules))
}

func (s *Storage) Held(holder bonuses.Holder) bonuses.Held {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.hands[holder]
}

func (s *Storage) SpendRefill(holder bonuses.Holder) bool {
	return s.spend(holder, bonuses.Held.AfterRefill)
}

func (s *Storage) SpendBomb(holder bonuses.Holder) bool {
	return s.spend(holder, bonuses.Held.AfterBomb)
}

func (s *Storage) SpendEnclose(holder bonuses.Holder) bool {
	return s.spend(holder, bonuses.Held.AfterEnclose)
}

func (s *Storage) SpendSpreadClick(holder bonuses.Holder) bool {
	return s.spend(holder, bonuses.Held.AfterSpreadClick)
}

func (s *Storage) spend(holder bonuses.Holder, after func(bonuses.Held) (bonuses.Held, bool)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	hand, ok := s.hands[holder]
	if !ok {
		return false
	}

	spent, ok := after(hand)
	if ok {
		s.putLocked(holder, spent)
	}

	return ok
}

func (s *Storage) putLocked(holder bonuses.Holder, hand bonuses.Held) {
	if hand.Empty() {
		delete(s.hands, holder)
	} else {
		s.hands[holder] = hand
	}
	s.dirty.Add(holder)
}

func (s *Storage) Name() string { return "charge-storage" }

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
		s.logger.Error("failed to flush the charges, retrying next tick", slog.Any("error", err))
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

	changed := make(map[bonuses.Holder]bonuses.Held, s.dirty.Len())
	s.dirty.ForEach(func(holder bonuses.Holder) { changed[holder] = s.hands[holder] })
	s.dirty = cpcolls.NewSet[bonuses.Holder]()
	s.mu.Unlock()

	if err := s.persistence.Save(ctx, changed); err != nil {
		s.mu.Lock()
		for holder := range changed {
			s.dirty.Add(holder)
		}
		s.mu.Unlock()

		return fmt.Errorf("failed to save %d hands: %w", len(changed), err)
	}

	return nil
}
