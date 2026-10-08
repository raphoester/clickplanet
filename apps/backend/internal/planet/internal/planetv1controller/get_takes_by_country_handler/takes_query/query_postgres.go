package takes_query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

// The tiles ledger.Taking replays for the account: a take's own row, and each tile a spread or an enclose took.
const takes = `
	WITH takes AS (
		SELECT country, previous FROM ledger_events
		WHERE account = $1 AND kind = 'take' AND country <> ''
		UNION ALL
		SELECT events.country, COALESCE(taken->>'owner', '')
		FROM ledger_events events, jsonb_array_elements(COALESCE(events.payload->'taken', '[]'::jsonb)) taken
		WHERE events.account = $1 AND events.kind IN ('spread', 'enclose') AND events.country <> ''
	)
	SELECT true, country, count(*) FROM takes GROUP BY country
	UNION ALL
	SELECT false, previous, count(*) FROM takes WHERE previous NOT IN ('', country) GROUP BY previous
	ORDER BY 3 DESC, 2
`

func (q *PostgresQuery) TakesByCountry(ctx context.Context, account cpsession.AccountID) (*planetv1.GetTakesByCountryResponse, error) {
	rows, err := q.db.QueryContext(ctx, takes, uuid.UUID(account))
	if err != nil {
		return nil, fmt.Errorf("failed to read the account's takes by country: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &planetv1.GetTakesByCountryResponse{}
	for rows.Next() {
		var (
			takenFor bool
			country  = &planetv1.CountryTakes{}
		)
		if err := rows.Scan(&takenFor, &country.CountryId, &country.Tiles); err != nil {
			return nil, fmt.Errorf("failed to read a country the account took tiles for or from: %w", err)
		}
		if takenFor {
			answer.TakenFor = append(answer.TakenFor, country)
		} else {
			answer.TakenFrom = append(answer.TakenFrom, country)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the account's takes by country: %w", err)
	}
	return answer, nil
}
