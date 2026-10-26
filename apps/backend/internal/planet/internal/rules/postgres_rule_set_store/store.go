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

func (s *Store) Set(ctx context.Context, name rules.SetName) (rules.Snapshot, error) {
	var kept []byte
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT rules FROM rule_sets WHERE name = $1`, string(name)).Scan(&kept)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return rules.Snapshot{}, fmt.Errorf("%q: %w", name, rules.ErrUnknownSet)
	case err != nil:
		return rules.Snapshot{}, fmt.Errorf("failed to read the rule set %q: %w", name, err)
	}

	var message planetv1.Rules
	if err := read.Unmarshal(kept, &message); err != nil {
		return rules.Snapshot{}, fmt.Errorf("the rule set %q kept is no planet.v1.Rules: %w", name, err)
	}
	return rules.Snapshot{Name: name, Rules: &message}, nil
}

func (s *Store) Save(ctx context.Context, set rules.Snapshot) error {
	document, err := written.Marshal(set.Rules)
	if err != nil {
		return fmt.Errorf("failed to write the rule set %q: %w", set.Name, err)
	}

	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO rule_sets (name, rules) VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET rules = EXCLUDED.rules
	`, string(set.Name), string(document)); err != nil {
		return fmt.Errorf("failed to keep the rule set %q: %w", set.Name, err)
	}

	return nil
}
