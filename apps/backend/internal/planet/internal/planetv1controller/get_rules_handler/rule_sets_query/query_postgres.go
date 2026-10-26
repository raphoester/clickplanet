package rule_sets_query

import (
	"context"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/rule_set_message"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

func (q *PostgresQuery) Sets(ctx context.Context) ([]*planetv1.RuleSet, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `SELECT name, rules FROM rule_sets ORDER BY name <> 'default', name`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rule sets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var sets []*planetv1.RuleSet
	for rows.Next() {
		var (
			name     string
			document []byte
		)
		if err := rows.Scan(&name, &document); err != nil {
			return nil, fmt.Errorf("failed to read a rule set: %w", err)
		}
		set, err := rule_set_message.Parse(name, document)
		if err != nil {
			return nil, err //nolint:wrapcheck // the codec names the set.
		}
		sets = append(sets, set)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rule sets: %w", err)
	}

	return sets, nil
}
