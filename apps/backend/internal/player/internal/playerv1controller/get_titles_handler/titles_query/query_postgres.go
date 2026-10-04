package titles_query

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Titles interface {
	Shown(held []string) []*playerv1.Title
	Worn(held []string, choice string) *playerv1.Title
	Tracks(held []string, career playerread.Career) []*playerv1.Track
}

func NewPostgresQuery(db cppg.Querier, titles Titles, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, titles: titles, clock: clock}
}

type PostgresQuery struct {
	db     cppg.Querier
	titles Titles
	clock  cptime.Clock
}

const career = `
	SELECT
		COALESCE(stats.tiles_taken, 0),
		CASE WHEN stats.streak_last_day IS NULL OR stats.streak_last_day IN ($2::date, $2::date - 1)
			THEN COALESCE(stats.streak_current, 0) ELSE 0 END,
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

func (q *PostgresQuery) Titles(ctx context.Context, account cpsession.AccountID) (*playerv1.GetTitlesResponse, error) {
	var (
		progress playerread.Career
		choice   string
		held     []string
	)
	if err := q.db.QueryRowContext(ctx, career, uuid.UUID(account), q.clock.Now().UTC().Format(time.DateOnly)).Scan(
		&progress.TilesTaken, &progress.StreakNow, &progress.MessagesSent, &choice, pq.Array(&held),
	); err != nil {
		return nil, fmt.Errorf("failed to read the titles: %w", err)
	}

	return &playerv1.GetTitlesResponse{
		Worn:     q.titles.Worn(held, choice),
		Wearable: q.titles.Shown(held),
		Tracks:   q.titles.Tracks(held, progress),
	}, nil
}
