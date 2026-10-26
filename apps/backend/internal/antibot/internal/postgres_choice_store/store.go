package postgres_choice_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
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
		set       string
		decidedAt time.Time
	)
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT rule_set, decided_at FROM rule_set_choice`).Scan(&set, &decidedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return rules.Choice{}, false, nil
	case err != nil:
		return rules.Choice{}, false, fmt.Errorf("failed to read the rule set chosen last: %w", err)
	}
	return rules.Choice{Set: rules.SetName(set), DecidedAt: decidedAt}, true, nil
}

func (s *Store) Save(ctx context.Context, choice rules.Choice) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO rule_set_choice (rule_set, decided_at) VALUES ($1, $2)
		ON CONFLICT (single) DO UPDATE SET rule_set = EXCLUDED.rule_set, decided_at = EXCLUDED.decided_at
	`, string(choice.Set), choice.DecidedAt.UTC()); err != nil {
		return fmt.Errorf("failed to keep the choice of the rule set %q: %w", choice.Set, err)
	}
	return nil
}
