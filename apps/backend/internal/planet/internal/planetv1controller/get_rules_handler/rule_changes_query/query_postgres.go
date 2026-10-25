package rule_changes_query

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const shown = 50

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

func (q *PostgresQuery) Changes(ctx context.Context) ([]*planetv1.RuleChange, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT request_id, rule_set, source::text, applied_at FROM rule_changes ORDER BY applied_at DESC, seq DESC LIMIT $1
	`, shown)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rule changes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var changes []*planetv1.RuleChange
	for rows.Next() {
		var (
			request, set, source string
			appliedAt            time.Time
		)
		if err := rows.Scan(&request, &set, &source, &appliedAt); err != nil {
			return nil, fmt.Errorf("failed to read a rule change: %w", err)
		}

		encoded, err := sourceOf(source)
		if err != nil {
			return nil, err
		}
		changes = append(changes, &planetv1.RuleChange{
			RequestId: request,
			RuleSet:   set,
			Source:    encoded,
			AppliedAt: timestamppb.New(appliedAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rule changes: %w", err)
	}

	return changes, nil
}

func sourceOf(source string) (planetv1.RuleChangeSource, error) {
	switch source {
	case "internal":
		return planetv1.RuleChangeSource_RULE_CHANGE_SOURCE_INTERNAL, nil
	case "operator":
		return planetv1.RuleChangeSource_RULE_CHANGE_SOURCE_OPERATOR, nil
	default:
		return planetv1.RuleChangeSource_RULE_CHANGE_SOURCE_UNSPECIFIED, fmt.Errorf("a rule change came from %q", source)
	}
}
