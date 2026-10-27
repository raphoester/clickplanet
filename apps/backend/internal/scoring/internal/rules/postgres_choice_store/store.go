package postgres_choice_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules"
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
		entry            uuid.NullUUID
		decidedAt        time.Time
	)
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT rule_set, entry_id, starts_at, ends_at, decided_at FROM rule_set_choice`).
		Scan(&set, &entry, &startsAt, &endsAt, &decidedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return rules.Choice{}, false, nil
	case err != nil:
		return rules.Choice{}, false, fmt.Errorf("failed to read the rule set chosen last: %w", err)
	}

	choice := rules.Choice{Set: rules.SetName(set), DecidedAt: decidedAt.UTC()}
	if endsAt.Valid {
		choice.Window = rules.Window{
			Entry: rounds.EntryID(entry.UUID), StartsAt: startsAt.Time.UTC(), EndsAt: endsAt.Time.UTC(),
		}
	}
	return choice, true, nil
}

func (s *Store) Save(ctx context.Context, choice rules.Choice) error {
	var (
		startsAt, endsAt sql.NullTime
		entry            uuid.NullUUID
	)
	if !choice.Window.None() {
		entry = uuid.NullUUID{UUID: uuid.UUID(choice.Window.Entry), Valid: true}
		startsAt = sql.NullTime{Time: choice.Window.StartsAt.UTC(), Valid: true}
		endsAt = sql.NullTime{Time: choice.Window.EndsAt.UTC(), Valid: true}
	}
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO rule_set_choice (rule_set, entry_id, starts_at, ends_at, decided_at) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (single) DO UPDATE
		SET rule_set = EXCLUDED.rule_set, entry_id = EXCLUDED.entry_id, starts_at = EXCLUDED.starts_at,
		    ends_at = EXCLUDED.ends_at, decided_at = EXCLUDED.decided_at
	`, string(choice.Set), entry, startsAt, endsAt, choice.DecidedAt.UTC()); err != nil {
		return fmt.Errorf("failed to keep the choice of the rule set %q: %w", choice.Set, err)
	}
	return nil
}
