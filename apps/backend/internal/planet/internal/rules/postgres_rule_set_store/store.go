package postgres_rule_set_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ rules.Store = (*Store)(nil)

func New(db cppg.Conn, parts rules.Parts) *Store {
	return &Store{db: db, parts: parts}
}

type Store struct {
	db    cppg.Conn
	parts rules.Parts
}

var (
	read    = protojson.UnmarshalOptions{DiscardUnknown: true}
	written = protojson.MarshalOptions{EmitUnpopulated: true}
)

func (s *Store) Set(ctx context.Context, name rules.SetName) (rules.Set, error) {
	var kept []byte
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT rules FROM rule_sets WHERE name = $1`, string(name)).Scan(&kept)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return rules.Set{}, fmt.Errorf("%q: %w", name, rules.ErrUnknownSet)
	case err != nil:
		return rules.Set{}, fmt.Errorf("failed to read the rule set %q: %w", name, err)
	}

	var message planetv1.Rules
	if err := read.Unmarshal(kept, &message); err != nil {
		return rules.Set{}, fmt.Errorf("the rule set %q kept is no planet.v1.Rules: %w", name, err)
	}

	set, err := rules.Snapshot{Name: name, Rules: &message}.Restore(s.parts)
	if err != nil {
		return rules.Set{}, fmt.Errorf("the rule set %q kept cannot play: %w", name, err)
	}
	return set, nil
}

func (s *Store) Save(ctx context.Context, set rules.Set) error {
	kept := set.TakeSnapshot()
	document, err := written.Marshal(kept.Rules)
	if err != nil {
		return fmt.Errorf("failed to write the rule set %q: %w", kept.Name, err)
	}

	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO rule_sets (name, rules) VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET rules = EXCLUDED.rules
	`, string(kept.Name), string(document)); err != nil {
		return fmt.Errorf("failed to keep the rule set %q: %w", kept.Name, err)
	}

	return nil
}
