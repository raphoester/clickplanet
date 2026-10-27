package rule_sets_query

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

var document = protojson.UnmarshalOptions{DiscardUnknown: true}

// Sets answers every set as postgres keeps it, one that cannot play too, so the operator can mend it.
func (q *PostgresQuery) Sets(ctx context.Context) ([]*seasonsv1.RuleSet, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `SELECT name, rules FROM rule_sets ORDER BY name <> 'default', name`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rule sets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var sets []*seasonsv1.RuleSet
	for rows.Next() {
		var (
			name string
			kept []byte
		)
		if err := rows.Scan(&name, &kept); err != nil {
			return nil, fmt.Errorf("failed to read a rule set: %w", err)
		}

		var rules seasonsv1.Rules
		if err := document.Unmarshal(kept, &rules); err != nil {
			return nil, fmt.Errorf("the rule set %q kept is no seasons.v1.Rules: %w", name, err)
		}
		sets = append(sets, &seasonsv1.RuleSet{Name: name, Rules: &rules})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rule sets: %w", err)
	}

	return sets, nil
}
