package authors_query

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

func NewPostgresQuery(db cppg.Querier, titles Titles, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, titles: titles, clock: clock}
}

type PostgresQuery struct {
	db     cppg.Querier
	titles Titles
	clock  cptime.Clock
}

const authors = `
	SELECT
		asked.account_id::text,
		COALESCE(profiles.name, 'guest_' || guest_codes.code),
		COALESCE(profiles.admin, false),
		COALESCE(profiles.color, 0),
		CASE WHEN profiles.account_id IS NULL THEN 0 ELSE ` + playerread.StreakNow + ` END,
		CASE WHEN profiles.account_id IS NULL THEN '' ELSE COALESCE(worn_titles.title, '') END,
		CASE WHEN profiles.account_id IS NULL THEN '{}' ELSE COALESCE(
			(SELECT array_agg(titles.title ORDER BY titles.earned_at, titles.title) FROM titles WHERE titles.account_id = asked.account_id),
			'{}'
		) END
	FROM (
		SELECT account_id, min(position) AS position
		FROM unnest($1::uuid[]) WITH ORDINALITY AS asked (account_id, position)
		GROUP BY account_id
	) asked
	LEFT JOIN profiles ON profiles.account_id = asked.account_id
	LEFT JOIN guest_codes ON guest_codes.account_id = asked.account_id
	LEFT JOIN stats ON stats.account_id = asked.account_id
	LEFT JOIN worn_titles ON worn_titles.account_id = asked.account_id
	WHERE profiles.account_id IS NOT NULL OR guest_codes.account_id IS NOT NULL
	ORDER BY asked.position
`

func (q *PostgresQuery) Authors(ctx context.Context, accounts []cpsession.AccountID) (*playerv1.GetAuthorsResponse, error) {
	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.String())
	}

	rows, err := q.db.QueryContext(ctx, authors, pq.Array(ids), q.clock.Now().UTC().Format(time.DateOnly))
	if err != nil {
		return nil, fmt.Errorf("failed to read the authors: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &playerv1.GetAuthorsResponse{Authors: make([]*playerv1.Author, 0, len(accounts))}
	for rows.Next() {
		var (
			author = &playerv1.Author{}
			color  int32
			choice string
			held   []string
		)
		if err := rows.Scan(
			&author.AccountId, &author.Name, &author.Admin, &color, &author.Streak, &choice, pq.Array(&held),
		); err != nil {
			return nil, fmt.Errorf("failed to scan an author: %w", err)
		}
		if author.Color, err = playerread.KeptColor(color); err != nil {
			return nil, fmt.Errorf("failed to read the color of %s: %w", author.GetAccountId(), err)
		}
		author.WornTitle = q.titles.Worn(held, choice)
		answer.Authors = append(answer.Authors, author)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the authors: %w", err)
	}
	return answer, nil
}
