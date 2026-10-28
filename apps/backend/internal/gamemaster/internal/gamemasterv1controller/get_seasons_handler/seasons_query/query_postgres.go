package seasons_query

import (
	"context"
	"fmt"
	"time"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/seasonmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

// Seasons is every season, by number, as planned and with the window season.Calendar works out.
func (q *PostgresQuery) Seasons(ctx context.Context) (*gamemasterv1.GetSeasonsResponse, error) {
	seasons, err := q.calendar(ctx)
	if err != nil {
		return nil, err
	}
	answer := &gamemasterv1.GetSeasonsResponse{}
	planned := seasons.TakeSnapshot()
	for i, window := range seasons.Windows() {
		answer.Seasons = append(answer.Seasons, seasonmessage.ScheduledOf(planned[i], window))
	}
	return answer, nil
}

func (q *PostgresQuery) calendar(ctx context.Context) (season.Calendar, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT number, (EXTRACT(EPOCH FROM break_before) * 1000000)::bigint, (EXTRACT(EPOCH FROM length) * 1000000)::bigint
		FROM seasons`)
	if err != nil {
		return season.Calendar{}, fmt.Errorf("failed to read the seasons: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var seasons []season.Season
	for rows.Next() {
		var number, brk, length int64
		if err := rows.Scan(&number, &brk, &length); err != nil {
			return season.Calendar{}, fmt.Errorf("failed to read a season: %w", err)
		}
		seasons = append(seasons, season.Snapshot{
			Number: season.Number(number), //nolint:gosec // CHECK (number >= 0 AND number <= 4294967295).
			Break:  time.Duration(brk) * time.Microsecond, Length: time.Duration(length) * time.Microsecond,
		}.Restore())
	}
	if err := rows.Err(); err != nil {
		return season.Calendar{}, fmt.Errorf("failed to read the seasons: %w", err)
	}
	return season.NewCalendar(seasons...), nil
}
