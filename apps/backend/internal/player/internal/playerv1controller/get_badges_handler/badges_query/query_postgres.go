package badges_query

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Titles interface {
	Worn(held []string, choice string) *playerv1.Title
}

func NewPostgresQuery(db cppg.Conn, titles Titles, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, titles: titles, clock: clock}
}

type PostgresQuery struct {
	db     cppg.Conn
	titles Titles
	clock  cptime.Clock
}

// Only a username wears a badge: a guest, with no profile, is left out, as an account never shown or deleted is.
const badges = `
	SELECT
		asked.account_id::text,
		` + playerread.StreakNow + `,
		COALESCE(worn_titles.title, ''),
		COALESCE(
			(SELECT array_agg(titles.title ORDER BY titles.earned_at, titles.title) FROM titles WHERE titles.account_id = asked.account_id),
			'{}'
		)
	FROM (
		SELECT account_id, min(position) AS position
		FROM unnest($1::uuid[]) WITH ORDINALITY AS asked (account_id, position)
		GROUP BY account_id
	) asked
	JOIN profiles ON profiles.account_id = asked.account_id
	LEFT JOIN stats ON stats.account_id = asked.account_id
	LEFT JOIN worn_titles ON worn_titles.account_id = asked.account_id
	-- An account its player asked to delete reads as deleted until a sign-in brings it back.
	WHERE NOT EXISTS (SELECT 1 FROM departures WHERE departures.account_id = asked.account_id AND departures.leaving)
	ORDER BY asked.position
`

func (q *PostgresQuery) Badges(ctx context.Context, accounts []cpsession.AccountID) (*playerv1.GetBadgesResponse, error) {
	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.String())
	}

	rows, err := q.db.On(ctx).QueryContext(ctx, badges, pq.Array(ids), q.clock.Now().UTC().Format(time.DateOnly))
	if err != nil {
		return nil, fmt.Errorf("failed to read the badges: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &playerv1.GetBadgesResponse{Badges: make([]*playerv1.Badge, 0, len(accounts))}
	for rows.Next() {
		var (
			badge  = &playerv1.Badge{}
			choice string
			held   []string
		)
		if err := rows.Scan(&badge.AccountId, &badge.Streak, &choice, pq.Array(&held)); err != nil {
			return nil, fmt.Errorf("failed to scan a badge: %w", err)
		}
		badge.WornTitle = q.titles.Worn(held, choice)
		if badge.GetStreak() == 0 && badge.GetWornTitle() == nil {
			continue
		}
		answer.Badges = append(answer.Badges, badge)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the badges: %w", err)
	}
	return answer, nil
}
