package cprules

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ Choices = (*PostgresChoices)(nil)

// PostgresChoices keeps the choice in the one row of rule_set_choice, in the schema db opens.
func NewPostgresChoices(db cppg.Conn) *PostgresChoices {
	return &PostgresChoices{db: db}
}

type PostgresChoices struct {
	db cppg.Conn
}

func (s *PostgresChoices) Choice(ctx context.Context) (Choice, bool, error) {
	var (
		set       string
		decidedAt time.Time
	)
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT rule_set, decided_at FROM rule_set_choice`).Scan(&set, &decidedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return Choice{}, false, nil
	case err != nil:
		return Choice{}, false, fmt.Errorf("failed to read the rule set chosen last: %w", err)
	}
	return Choice{Set: SetName(set), DecidedAt: decidedAt}, true, nil
}

func (s *PostgresChoices) Save(ctx context.Context, choice Choice) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO rule_set_choice (rule_set, decided_at) VALUES ($1, $2)
		ON CONFLICT (single) DO UPDATE SET rule_set = EXCLUDED.rule_set, decided_at = EXCLUDED.decided_at
	`, string(choice.Set), choice.DecidedAt.UTC()); err != nil {
		return fmt.Errorf("failed to keep the choice of the rule set %q: %w", choice.Set, err)
	}
	return nil
}
