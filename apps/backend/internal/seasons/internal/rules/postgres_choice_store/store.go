package postgres_choice_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ rules.Choices = (*Store)(nil)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

func (s *Store) Choice(ctx context.Context) (rules.Choice, bool, error) {
	var (
		set              string
		startsAt, endsAt sql.NullTime
		decidedAt        time.Time
	)
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT rule_set, starts_at, ends_at, decided_at FROM rule_set_choice`).
		Scan(&set, &startsAt, &endsAt, &decidedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return rules.Choice{}, false, nil
	case err != nil:
		return rules.Choice{}, false, fmt.Errorf("failed to read the rule set chosen last: %w", err)
	}

	choice := rules.Choice{Set: rules.SetName(set), DecidedAt: decidedAt.UTC()}
	if endsAt.Valid {
		choice.Window = rules.Window{StartsAt: startsAt.Time.UTC(), EndsAt: endsAt.Time.UTC()}
	}
	return choice, true, nil
}

func (s *Store) Save(ctx context.Context, choice rules.Choice) error {
	var startsAt, endsAt sql.NullTime
	if !choice.Window.None() {
		startsAt = sql.NullTime{Time: choice.Window.StartsAt.UTC(), Valid: true}
		endsAt = sql.NullTime{Time: choice.Window.EndsAt.UTC(), Valid: true}
	}
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO rule_set_choice (rule_set, starts_at, ends_at, decided_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT (single) DO UPDATE
		SET rule_set = EXCLUDED.rule_set, starts_at = EXCLUDED.starts_at, ends_at = EXCLUDED.ends_at,
		    decided_at = EXCLUDED.decided_at
	`, string(choice.Set), startsAt, endsAt, choice.DecidedAt.UTC()); err != nil {
		return fmt.Errorf("failed to keep the choice of the rule set %q: %w", choice.Set, err)
	}
	return nil
}
