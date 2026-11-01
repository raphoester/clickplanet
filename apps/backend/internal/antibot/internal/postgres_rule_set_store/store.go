package postgres_rule_set_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
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

var read = protojson.UnmarshalOptions{DiscardUnknown: true}

func (s *Store) Set(ctx context.Context, name rules.SetName) (rules.Set, error) {
	defaults, err := s.kept(ctx, rules.Default)
	if err != nil {
		return rules.Set{}, err
	}
	kept, err := s.kept(ctx, name)
	if err != nil {
		return rules.Set{}, err
	}

	set, err := kept.Restore(s.parts, defaults)
	if err != nil {
		return rules.Set{}, fmt.Errorf("the rule set %q kept cannot play: %w", name, err)
	}
	return set, nil
}

func (s *Store) kept(ctx context.Context, name rules.SetName) (rules.Snapshot, error) {
	var kept []byte
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT rules FROM rule_sets WHERE name = $1`, string(name)).Scan(&kept)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return rules.Snapshot{}, fmt.Errorf("%q: %w", name, rules.ErrUnknownSet)
	case err != nil:
		return rules.Snapshot{}, fmt.Errorf("failed to read the rule set %q: %w", name, err)
	}

	var message antibotv1.Rules
	if err := read.Unmarshal(kept, &message); err != nil {
		return rules.Snapshot{}, fmt.Errorf("the rule set %q kept is no antibot.v1.Rules: %w", name, err)
	}
	return rules.Snapshot{Name: name, Rules: &message}, nil
}

func (s *Store) Names(ctx context.Context) ([]rules.SetName, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT name FROM rule_sets ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rule set names: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var names []rules.SetName
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to read a rule set name: %w", err)
		}
		names = append(names, rules.SetName(name))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rule set names: %w", err)
	}
	return names, nil
}

func (s *Store) Save(ctx context.Context, set rules.Set) error {
	kept := set.TakeSnapshot()
	document, err := protojson.Marshal(kept.Rules)
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
