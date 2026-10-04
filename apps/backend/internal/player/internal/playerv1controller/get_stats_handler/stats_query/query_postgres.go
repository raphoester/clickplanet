package stats_query

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewPostgresQuery(db cppg.Querier, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock}
}

type PostgresQuery struct {
	db    cppg.Querier
	clock cptime.Clock
}

const stats = `
	SELECT
		COALESCE(stats.tiles_taken, 0),
		CASE WHEN stats.streak_last_day IS NULL OR stats.streak_last_day IN ($2::date, $2::date - 1)
			THEN COALESCE(stats.streak_current, 0) ELSE 0 END,
		COALESCE(stats.streak_best, 0),
		COALESCE(to_char(stats.streak_last_day, 'YYYY-MM-DD'), '')
	FROM (SELECT $1::uuid AS account_id) asked
	LEFT JOIN stats ON stats.account_id = asked.account_id
`

func (q *PostgresQuery) Stats(ctx context.Context, account cpsession.AccountID) (*playerv1.GetStatsResponse, error) {
	answer := &playerv1.Stats{}
	if err := q.db.QueryRowContext(ctx, stats, uuid.UUID(account), q.clock.Now().UTC().Format(time.DateOnly)).
		Scan(&answer.TilesTaken, &answer.StreakCurrent, &answer.StreakBest, &answer.StreakLastDay); err != nil {
		return nil, fmt.Errorf("failed to read the stats: %w", err)
	}
	return &playerv1.GetStatsResponse{Stats: answer}, nil
}
