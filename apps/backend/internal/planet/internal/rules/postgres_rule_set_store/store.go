package postgres_rule_set_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ rules.Store = (*Store)(nil)

func (s *Store) Set(ctx context.Context, name rules.SetName) (rules.Set, error) {
	var (
		kept                     rules.Snapshot
		minInterval, maxInterval int64
	)
	err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT name, tempo, box_min_interval_ms, box_max_interval_ms, box_max_charges_per_hour, crowd
		FROM rule_sets WHERE name = $1
	`, string(name)).Scan(&kept.Name, &kept.Tempo, &minInterval, &maxInterval, &kept.Boxes.MaxChargesPerHour, &kept.Crowd)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return rules.Set{}, fmt.Errorf("%q: %w", name, rules.ErrUnknownSet)
	case err != nil:
		return rules.Set{}, fmt.Errorf("failed to read the rule set %q: %w", name, err)
	}

	kept.Boxes = bonuses.Pace{
		MinInterval:       time.Duration(minInterval) * time.Millisecond,
		MaxInterval:       time.Duration(maxInterval) * time.Millisecond,
		MaxChargesPerHour: kept.Boxes.MaxChargesPerHour,
	}
	set, err := kept.Restore()
	if err != nil {
		return rules.Set{}, fmt.Errorf("the rule set %q kept cannot play: %w", name, err)
	}
	return set, nil
}

func (s *Store) Save(ctx context.Context, set rules.Set) error {
	kept := set.TakeSnapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO rule_sets (name, tempo, box_min_interval_ms, box_max_interval_ms, box_max_charges_per_hour, crowd)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (name) DO UPDATE SET tempo = EXCLUDED.tempo, box_min_interval_ms = EXCLUDED.box_min_interval_ms,
			box_max_interval_ms = EXCLUDED.box_max_interval_ms,
			box_max_charges_per_hour = EXCLUDED.box_max_charges_per_hour, crowd = EXCLUDED.crowd
	`, string(kept.Name), kept.Tempo, kept.Boxes.MinInterval.Milliseconds(), kept.Boxes.MaxInterval.Milliseconds(),
		kept.Boxes.MaxChargesPerHour, kept.Crowd); err != nil {
		return fmt.Errorf("failed to keep the rule set %q: %w", kept.Name, err)
	}

	return nil
}
