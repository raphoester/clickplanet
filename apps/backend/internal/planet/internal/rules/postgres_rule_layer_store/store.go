package postgres_rule_layer_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ rules.LayerStore = (*Store)(nil)

func (s *Store) Baseline(ctx context.Context, request rules.RequestID) (rules.Baseline, error) {
	return s.baseline(ctx, `
		SELECT request, rule_set, applied_at FROM rule_baselines WHERE request = $1
	`, string(request))
}

func (s *Store) LatestBaseline(ctx context.Context) (rules.Baseline, error) {
	return s.baseline(ctx, `
		SELECT request, rule_set, applied_at FROM rule_baselines ORDER BY applied_at DESC, seq DESC LIMIT 1
	`)
}

func (s *Store) baseline(ctx context.Context, query string, args ...any) (rules.Baseline, error) {
	var kept rules.BaselineSnapshot
	err := s.db.On(ctx).QueryRowContext(ctx, query, args...).Scan(&kept.Request, &kept.Set, &kept.AppliedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return rules.Baseline{}, nil
	case err != nil:
		return rules.Baseline{}, fmt.Errorf("failed to read a baseline: %w", err)
	}

	return kept.Restore(), nil
}

func (s *Store) SaveBaseline(ctx context.Context, baseline rules.Baseline) error {
	kept := baseline.TakeSnapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO rule_baselines (request, rule_set, applied_at) VALUES ($1, $2, $3)
		ON CONFLICT (request) DO NOTHING
	`, string(kept.Request), string(kept.Set), kept.AppliedAt); err != nil {
		return fmt.Errorf("failed to keep the baseline of request %q: %w", kept.Request, err)
	}

	return nil
}

func (s *Store) Override(ctx context.Context) (rules.Override, error) {
	var kept rules.OverrideSnapshot
	err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT rule_set, starts_at, ends_at FROM rule_override
	`).Scan(&kept.Set, &kept.StartsAt, &kept.EndsAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return rules.Override{}, nil
	case err != nil:
		return rules.Override{}, fmt.Errorf("failed to read the override: %w", err)
	}

	return kept.Restore(), nil
}

func (s *Store) SaveOverride(ctx context.Context, override rules.Override) error {
	kept := override.TakeSnapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO rule_override (rule_set, starts_at, ends_at) VALUES ($1, $2, $3)
		ON CONFLICT (only_one) DO UPDATE SET rule_set = EXCLUDED.rule_set, starts_at = EXCLUDED.starts_at,
			ends_at = EXCLUDED.ends_at
	`, string(kept.Set), kept.StartsAt, kept.EndsAt); err != nil {
		return fmt.Errorf("failed to keep the override: %w", err)
	}

	return nil
}
