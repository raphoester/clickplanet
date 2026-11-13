package rule_set_names_query

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

func (q *PostgresQuery) Names(ctx context.Context) ([]string, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `SELECT name FROM rule_sets ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rule set names: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to read a rule set name: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rule set names: %w", err)
	}
	return names, nil
}
