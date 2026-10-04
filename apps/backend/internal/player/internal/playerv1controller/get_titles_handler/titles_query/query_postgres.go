package titles_query

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewPostgresQuery(db cppg.Querier, catalog titles.Catalog, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, catalog: catalog, clock: clock}
}

type PostgresQuery struct {
	db      cppg.Querier
	catalog titles.Catalog
	clock   cptime.Clock
}

const career = `
	SELECT
		COALESCE(stats.tiles_taken, 0),
		COALESCE(stats.streak_current, 0),
		COALESCE(stats.streak_best, 0),
		stats.streak_last_day,
		COALESCE(stats.messages_sent, 0),
		COALESCE(worn_titles.title, ''),
		COALESCE(
			(SELECT array_agg(titles.title ORDER BY titles.earned_at, titles.title) FROM titles WHERE titles.account_id = asked.account_id),
			'{}'
		)
	FROM (SELECT $1::uuid AS account_id) asked
	LEFT JOIN stats ON stats.account_id = asked.account_id
	LEFT JOIN worn_titles ON worn_titles.account_id = asked.account_id
`

func (q *PostgresQuery) Titles(ctx context.Context, account players.AccountID) (*playerv1.GetTitlesResponse, error) {
	var (
		stats   = players.Stats{Account: account}
		lastDay sql.NullTime
		choice  string
		owned   []string
	)
	if err := q.db.QueryRowContext(ctx, career, uuid.UUID(account)).Scan(
		&stats.TilesTaken, &stats.StreakCurrent, &stats.StreakBest, &lastDay, &stats.MessagesSent,
		&choice, pq.Array(&owned),
	); err != nil {
		return nil, fmt.Errorf("failed to read the titles: %w", err)
	}
	if lastDay.Valid {
		stats.StreakLastDay = players.DayOf(lastDay.Time)
	}

	held := make(titles.IDs, 0, len(owned))
	for _, id := range owned {
		held = append(held, titles.ID(id))
	}

	tracks := q.catalog.Progress(titles.Career{Stats: stats.AsOf(players.DayOf(q.clock.Now()))}, held)
	return dashboardOf(wearing.ShowcaseOf(q.catalog, held, titles.ID(choice)), tracks), nil
}

func dashboardOf(showcase wearing.Showcase, progress []titles.TrackProgress) *playerv1.GetTitlesResponse {
	tracks := make([]*playerv1.Track, 0, len(progress))
	for _, track := range progress {
		steps := make([]*playerv1.Step, 0, len(track.Steps))
		for _, step := range track.Steps {
			steps = append(steps, &playerv1.Step{Title: playermessage.Title(step.Standing), Threshold: step.Threshold, Earned: step.Earned})
		}
		tracks = append(tracks, &playerv1.Track{Id: string(track.ID), Name: track.Name, Progress: track.Progress, Steps: steps})
	}
	return &playerv1.GetTitlesResponse{
		Worn:     playermessage.Title(showcase.Worn),
		Wearable: playermessage.Titles(showcase.Shown),
		Tracks:   tracks,
	}
}
