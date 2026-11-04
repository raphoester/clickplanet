package progress_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/progressionread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Titles interface {
	Shown(held []string) []*progressionv1.Title
	Worn(held []string, choice string) *progressionv1.Title
}

type Fronts interface {
	Fronts(ctx context.Context, account accounts.AccountID) (*progressionv1.GetFrontsResponse, error)
}

var ErrNotShown = errors.New("the other players do not see this account")

const shownFronts = 3

func NewPostgresQuery(db cppg.Conn, titles Titles, fronts Fronts, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, titles: titles, fronts: fronts, clock: clock}
}

type PostgresQuery struct {
	db     cppg.Conn
	titles Titles
	fronts Fronts
	clock  cptime.Clock
}

// Only a username has a card: a guest shows no streak and wears no title anywhere.
const progress = `
	SELECT
		COALESCE(stats.tiles_taken, 0),
		` + progressionread.StreakNow + `,
		COALESCE(stats.streak_best, 0),
		COALESCE(to_char(stats.streak_last_day, 'YYYY-MM-DD'), ''),
		COALESCE(worn_titles.title, ''),
		COALESCE(
			(SELECT array_agg(titles.title ORDER BY titles.earned_at, titles.title) FROM titles WHERE titles.account_id = asked.account_id),
			'{}'
		)
	FROM (SELECT $1::uuid AS account_id) asked
	JOIN nameplates ON nameplates.account_id = asked.account_id
	LEFT JOIN stats ON stats.account_id = asked.account_id
	LEFT JOIN worn_titles ON worn_titles.account_id = asked.account_id
	-- An account its player asked to delete reads as deleted until a sign-in brings it back.
	WHERE NOT EXISTS (SELECT 1 FROM departures WHERE departures.account_id = asked.account_id AND departures.leaving)
`

func (q *PostgresQuery) Progress(ctx context.Context, account accounts.AccountID) (*progressionv1.GetProgressResponse, error) {
	var (
		choice string
		held   []string
		answer = &progressionv1.Progress{Stats: &progressionv1.Stats{}}
	)
	err := q.db.On(ctx).QueryRowContext(ctx, progress, uuid.UUID(account), q.clock.Now().UTC().Format(time.DateOnly)).Scan(
		&answer.Stats.TilesTaken, &answer.Stats.StreakCurrent, &answer.Stats.StreakBest, &answer.Stats.StreakLastDay,
		&choice, pq.Array(&held),
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotShown
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the progress: %w", err)
	}

	fronts, err := q.fronts.Fronts(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to ask which countries the player plays for and against: %w", err)
	}
	answer.PlaysFor, answer.PlaysAgainst = topOf(fronts.GetPlaysFor()), topOf(fronts.GetPlaysAgainst())
	answer.Titles = q.titles.Shown(held)
	answer.WornTitle = q.titles.Worn(held, choice)
	return &progressionv1.GetProgressResponse{Progress: answer}, nil
}

func topOf(countries []*progressionv1.CountryTiles) []*progressionv1.CountryTiles {
	return countries[:min(len(countries), shownFronts)]
}
