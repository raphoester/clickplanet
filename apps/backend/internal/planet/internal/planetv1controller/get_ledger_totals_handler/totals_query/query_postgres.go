package totals_query

import (
	"context"
	"fmt"
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

const totals = `
	SELECT
		count(*) FILTER (WHERE kind = 'take'),
		count(*) FILTER (WHERE kind = 'strike'),
		count(*) FILTER (WHERE kind = 'spread'),
		count(*) FILTER (WHERE kind = 'enclose'),
		count(*) FILTER (WHERE kind = 'bomb'),
		count(*) FILTER (WHERE kind = 'shield'),
		count(*) FILTER (WHERE kind = 'fortify'),
		count(DISTINCT account)
	FROM ledger_events
	WHERE taken_at >= $1 AND taken_at < $2
`

func (q *PostgresQuery) Totals(ctx context.Context, since, until time.Time) (*planetv1.LedgerTotals, error) {
	answer := &planetv1.LedgerTotals{}
	err := q.db.On(ctx).QueryRowContext(ctx, totals, since, until).Scan(
		&answer.Takes, &answer.Strikes, &answer.Spreads, &answer.Encloses, &answer.Bombs, &answer.Shields, &answer.Fortifies,
		&answer.Accounts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to count the ledger's events from %s to %s: %w", since, until, err)
	}
	return answer, nil
}
