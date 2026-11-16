package play_days_query

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

// The UTC days of the events the relay tells as TilesTaken: a take's own row, and a spread, an enclose or a virus that
// took a tile. A bomb's and a knock-out's tiles are taken for nobody, and a strike, a shield or a fortify takes none. A
// reverted event still counts, as the stats never take a tile back.
const playDays = `
	SELECT account, array_agg(
		DISTINCT to_char(taken_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') ORDER BY to_char(taken_at AT TIME ZONE 'UTC', 'YYYY-MM-DD')
	)
	FROM ledger_events
	WHERE account = ANY($1::uuid[]) AND kind IN ('take', 'spread', 'enclose', 'virus') AND country <> ''
		AND (kind = 'take' OR jsonb_array_length(COALESCE(payload->'taken', '[]'::jsonb)) > 0)
	GROUP BY account
	ORDER BY account
`

func (q *PostgresQuery) PlayDays(ctx context.Context, accounts []cpsession.AccountID) (*planetv1.GetPlayDaysResponse, error) {
	ids := make([]string, len(accounts))
	for i, account := range accounts {
		ids[i] = account.String()
	}
	rows, err := q.db.On(ctx).QueryContext(ctx, playDays, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to read the days the accounts took a tile on: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &planetv1.GetPlayDaysResponse{}
	for rows.Next() {
		var (
			account uuid.UUID
			played  = &planetv1.AccountPlayDays{}
		)
		if err := rows.Scan(&account, pq.Array(&played.Days)); err != nil {
			return nil, fmt.Errorf("failed to read the days an account took a tile on: %w", err)
		}
		played.AccountId = account.String()
		answer.Accounts = append(answer.Accounts, played)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the days the accounts took a tile on: %w", err)
	}
	return answer, nil
}
