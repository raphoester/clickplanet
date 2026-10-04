package feed_start_query

import (
	"context"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

func (q *PostgresQuery) Start(ctx context.Context) (*planetv1.GetFeedStartResponse, error) {
	answer := &planetv1.GetFeedStartResponse{}
	if err := q.db.QueryRowContext(ctx, `SELECT start FROM ledger_feed`).Scan(&answer.Position); err != nil {
		return nil, fmt.Errorf("failed to read where the feed starts: %w", err)
	}
	return answer, nil
}
