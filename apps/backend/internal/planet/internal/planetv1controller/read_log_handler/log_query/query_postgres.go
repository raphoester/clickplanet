package log_query

import (
	"context"
	"fmt"
	"math"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const MaxEntries = 1000

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

const entries = `
	SELECT position, tile, COALESCE(account::text, ''), country, previous, taken_at, reverted
	FROM ledger_takes
	WHERE position >= $1
	ORDER BY position
	LIMIT $2
`

func (q *PostgresQuery) Entries(ctx context.Context, from uint64, limit uint32) (*planetv1.ReadLogResponse, error) {
	if limit == 0 || limit > MaxEntries {
		limit = MaxEntries
	}

	rows, err := q.db.QueryContext(ctx, entries, int64(min(from, math.MaxInt64)), limit) //nolint:gosec // capped just above.
	if err != nil {
		return nil, fmt.Errorf("failed to read the log: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &planetv1.ReadLogResponse{}
	for rows.Next() {
		var (
			position uint64
			take     = &planetv1.Take{}
			at       time.Time
		)
		if err := rows.Scan(&position, &take.TileId, &take.AccountId, &take.Country, &take.Previous, &at, &take.Reverted); err != nil {
			return nil, fmt.Errorf("failed to read an entry of the log: %w", err)
		}
		take.TakenAt = timestamppb.New(at)
		answer.Entries = append(answer.Entries, &planetv1.LogEntry{Position: position, Fact: &planetv1.LogEntry_Take{Take: take}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the log: %w", err)
	}

	return answer, nil
}
