package cprules

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// PostgresStore keeps each set as one row of rule_sets, in the schema db opens: its name, and the values it holds as
// protojson writes an M.
func NewPostgresStore[M proto.Message](db cppg.Conn, parts Parts[M]) *PostgresStore[M] {
	return &PostgresStore[M]{db: db, parts: parts}
}

type PostgresStore[M proto.Message] struct {
	db    cppg.Conn
	parts Parts[M]
}

// A key the message no longer has is left alone: planet's sets still hold the offers' values under bonuses.
var read = protojson.UnmarshalOptions{DiscardUnknown: true}

func (s *PostgresStore[M]) Set(ctx context.Context, name SetName) (Set[M], error) {
	defaults, err := s.kept(ctx, Default)
	if err != nil {
		return Set[M]{}, err
	}
	kept, err := s.kept(ctx, name)
	if err != nil {
		return Set[M]{}, err
	}

	set, err := kept.Restore(s.parts, defaults)
	if err != nil {
		return Set[M]{}, fmt.Errorf("the rule set %q kept cannot play: %w", name, err)
	}
	return set, nil
}

func (s *PostgresStore[M]) kept(ctx context.Context, name SetName) (Snapshot[M], error) {
	var kept []byte
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT rules FROM rule_sets WHERE name = $1`, string(name)).Scan(&kept)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return Snapshot[M]{}, fmt.Errorf("%q: %w", name, ErrUnknownSet)
	case err != nil:
		return Snapshot[M]{}, fmt.Errorf("failed to read the rule set %q: %w", name, err)
	}

	message := empty[M]()
	if err := read.Unmarshal(kept, message); err != nil {
		return Snapshot[M]{}, fmt.Errorf("the rule set %q kept is no %s: %w",
			name, message.ProtoReflect().Descriptor().FullName(), err)
	}
	return Snapshot[M]{Name: name, Rules: message}, nil
}

func (s *PostgresStore[M]) Names(ctx context.Context) ([]SetName, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT name FROM rule_sets ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rule set names: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var names []SetName
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to read a rule set name: %w", err)
		}
		names = append(names, SetName(name))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rule set names: %w", err)
	}
	return names, nil
}

func (s *PostgresStore[M]) Save(ctx context.Context, set Set[M]) error {
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
