package postgres_rule_set_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules/rulesmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ rules.Store = (*Store)(nil)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

// A key a later version no longer reads is left alone: the set still restores, or says what it lacks.
var document = protojson.UnmarshalOptions{DiscardUnknown: true}

func (s *Store) Set(ctx context.Context, name rules.SetName) (rules.Set, error) {
	var kept []byte
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT rules FROM rule_sets WHERE name = $1`, string(name)).Scan(&kept)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return rules.Set{}, fmt.Errorf("%q: %w", name, rules.ErrUnknownSet)
	case err != nil:
		return rules.Set{}, fmt.Errorf("failed to read the rule set %q: %w", name, err)
	}

	var message antibotv1.Rules
	if err := document.Unmarshal(kept, &message); err != nil {
		return rules.Set{}, fmt.Errorf("the rule set %q kept is no antibot.v1.Rules: %w", name, err)
	}

	set, err := rulesmessage.SnapshotOf(name, &message).Restore()
	if err != nil {
		return rules.Set{}, fmt.Errorf("the rule set %q kept cannot play: %w", name, err)
	}
	return set, nil
}

func (s *Store) Save(ctx context.Context, set rules.Set) error {
	kept := set.TakeSnapshot()

	written, err := rulesmessage.Document.Marshal(rulesmessage.Of(kept))
	if err != nil {
		return fmt.Errorf("failed to write the rule set %q: %w", kept.Name, err)
	}

	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO rule_sets (name, rules) VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET rules = EXCLUDED.rules
	`, string(kept.Name), string(written)); err != nil {
		return fmt.Errorf("failed to keep the rule set %q: %w", kept.Name, err)
	}

	return nil
}
