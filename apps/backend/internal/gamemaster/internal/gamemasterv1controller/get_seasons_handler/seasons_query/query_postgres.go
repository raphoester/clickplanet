package seasons_query

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/seasonmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
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
		SELECT s.number, (EXTRACT(EPOCH FROM s.break_before) * 1000000)::bigint, (EXTRACT(EPOCH FROM s.length) * 1000000)::bigint,
			(EXTRACT(EPOCH FROM f.length) * 1000000)::bigint, f.rule_set, f.description, f.times
		FROM seasons AS s LEFT JOIN season_finales AS f ON f.season = s.number`)
	if err != nil {
		return season.Calendar{}, fmt.Errorf("failed to read the seasons: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var seasons []season.Season
	for rows.Next() {
		var (
			number, brk, length       int64
			finaleLength, times       sql.NullInt64
			finaleSet, finaleDescribe sql.NullString
		)
		if err := rows.Scan(&number, &brk, &length, &finaleLength, &finaleSet, &finaleDescribe, &times); err != nil {
			return season.Calendar{}, fmt.Errorf("failed to read a season: %w", err)
		}
		planned := season.Snapshot{
			Number: season.Number(number), //nolint:gosec // CHECK (number >= 0 AND number <= 4294967295).
			Break:  time.Duration(brk) * time.Microsecond, Length: time.Duration(length) * time.Microsecond,
		}
		if finaleLength.Valid {
			planned.Finale = season.Finale{
				Length: time.Duration(finaleLength.Int64) * time.Microsecond, Set: schedule.SetName(finaleSet.String),
				Description: schedule.Description(finaleDescribe.String),
				Times:       schedule.Times(times.Int64), //nolint:gosec // CHECK (times BETWEEN 1 AND 10).
			}
		}
		seasons = append(seasons, planned.Restore())
	}
	if err := rows.Err(); err != nil {
		return season.Calendar{}, fmt.Errorf("failed to read the seasons: %w", err)
	}
	return season.NewCalendar(seasons...), nil
}
