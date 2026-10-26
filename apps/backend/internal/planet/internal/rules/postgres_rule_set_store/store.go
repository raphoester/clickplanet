package postgres_rule_set_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/rule_set_message"
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
	var document []byte
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT rules FROM rule_sets WHERE name = $1`, string(name)).Scan(&document)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return rules.Set{}, fmt.Errorf("%q: %w", name, rules.ErrUnknownSet)
	case err != nil:
		return rules.Set{}, fmt.Errorf("failed to read the rule set %q: %w", name, err)
	}

	message, err := rule_set_message.Parse(string(name), document)
	if err != nil {
		return rules.Set{}, err //nolint:wrapcheck // the codec names the set.
	}
	set, err := rule_set_message.Decode(message).Restore()
	if err != nil {
		return rules.Set{}, fmt.Errorf("the rule set %q kept cannot play: %w", name, err)
	}
	return set, nil
}

func (s *Store) Save(ctx context.Context, set rules.Set) error {
	kept := set.TakeSnapshot()
	document, err := rule_set_message.Document(kept)
	if err != nil {
		return err //nolint:wrapcheck // the codec names the set.
	}

	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO rule_sets (name, rules) VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET rules = EXCLUDED.rules
	`, string(kept.Name), document); err != nil {
		return fmt.Errorf("failed to keep the rule set %q: %w", kept.Name, err)
	}

	return nil
}
