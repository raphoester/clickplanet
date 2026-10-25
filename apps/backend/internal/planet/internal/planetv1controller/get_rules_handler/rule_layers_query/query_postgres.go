package rule_layers_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const shownBaselines = 20

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

func (q *PostgresQuery) Baselines(ctx context.Context) ([]*planetv1.RuleBaseline, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT request, rule_set, applied_at FROM rule_baselines ORDER BY applied_at DESC, seq DESC LIMIT $1
	`, shownBaselines)
	if err != nil {
		return nil, fmt.Errorf("failed to read the baselines: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var baselines []*planetv1.RuleBaseline
	for rows.Next() {
		var (
			request, set string
			appliedAt    time.Time
		)
		if err := rows.Scan(&request, &set, &appliedAt); err != nil {
			return nil, fmt.Errorf("failed to read a baseline: %w", err)
		}
		baselines = append(baselines, &planetv1.RuleBaseline{
			RequestId: request, RuleSet: set, AppliedAt: timestamppb.New(appliedAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the baselines: %w", err)
	}

	return baselines, nil
}

func (q *PostgresQuery) Override(ctx context.Context) (*planetv1.RuleOverride, error) {
	var (
		set          string
		starts, ends time.Time
	)
	err := q.db.On(ctx).QueryRowContext(ctx, `SELECT rule_set, starts_at, ends_at FROM rule_override`).Scan(&set, &starts, &ends)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil //nolint:nilnil // no override ever set is no message on the wire.
	case err != nil:
		return nil, fmt.Errorf("failed to read the override: %w", err)
	}

	return &planetv1.RuleOverride{RuleSet: set, StartsAt: timestamppb.New(starts), EndsAt: timestamppb.New(ends)}, nil
}
