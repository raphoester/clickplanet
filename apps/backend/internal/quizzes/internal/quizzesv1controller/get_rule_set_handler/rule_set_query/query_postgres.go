package rule_set_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpproto"
)

var ErrNoSet = errors.New("no rule set has this name")

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

// A value a newer build wrote is left out rather than refused: the caller fills words with the values it knows.
var read = protojson.UnmarshalOptions{DiscardUnknown: true}

// Played is default's values with the set's own written over them, as the set plays. Default plays alone.
func (q *PostgresQuery) Played(ctx context.Context, name string) (*quizzesv1.Rules, error) {
	var defaults, own sql.NullString
	err := q.db.On(ctx).QueryRowContext(ctx, `
		SELECT d.rules, s.rules
		FROM rule_sets d
		LEFT JOIN rule_sets s ON s.name = $1
		WHERE d.name = 'default'`, name).Scan(&defaults, &own)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, errors.New("no default rule set is kept")
	case err != nil:
		return nil, fmt.Errorf("failed to read the rule set %q: %w", name, err)
	case !own.Valid:
		return nil, fmt.Errorf("%w: %q", ErrNoSet, name)
	}

	base, err := decode(defaults.String)
	if err != nil {
		return nil, fmt.Errorf("failed to read the default rule set: %w", err)
	}
	top, err := decode(own.String)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rule set %q: %w", name, err)
	}
	return cpproto.Overlay(base, top), nil
}

func decode(document string) (*quizzesv1.Rules, error) {
	rules := &quizzesv1.Rules{}
	if err := read.Unmarshal([]byte(document), rules); err != nil {
		return nil, fmt.Errorf("failed to decode the rules: %w", err)
	}
	return rules, nil
}
