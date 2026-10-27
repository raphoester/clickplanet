package postgres_rule_set_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ rules.Store = (*Store)(nil)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
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

	var message seasonsv1.Rules
	if err := read.Unmarshal(kept, &message); err != nil {
		return rules.Set{}, fmt.Errorf("the rule set %q kept is no seasons.v1.Rules: %w", name, err)
	}

	set, err := rules.Snapshot{Name: name, Rules: &message}.Restore()
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
