package stats_query

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
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
		COALESCE(stats.streak_current, 0),
		COALESCE(stats.streak_best, 0),
		stats.streak_last_day
	FROM (SELECT $1::uuid AS account_id) asked
	LEFT JOIN stats ON stats.account_id = asked.account_id
`

func (q *PostgresQuery) Stats(ctx context.Context, account players.AccountID) (*playerv1.GetStatsResponse, error) {
	kept := players.Stats{Account: account}
	var lastDay sql.NullTime
	if err := q.db.QueryRowContext(ctx, stats, uuid.UUID(account)).
		Scan(&kept.TilesTaken, &kept.StreakCurrent, &kept.StreakBest, &lastDay); err != nil {
		return nil, fmt.Errorf("failed to read the stats: %w", err)
	}
	if lastDay.Valid {
		kept.StreakLastDay = players.DayOf(lastDay.Time)
	}

	return &playerv1.GetStatsResponse{Stats: playermessage.Stats(kept.AsOf(players.DayOf(q.clock.Now())))}, nil
}
