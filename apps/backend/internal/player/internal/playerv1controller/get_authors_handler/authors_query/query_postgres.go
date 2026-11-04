package authors_query

import (
	"context"
	"fmt"

	"github.com/lib/pq"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

const authors = `
	SELECT
		asked.account_id::text,
		COALESCE(profiles.name, 'guest_' || guest_codes.code),
		COALESCE(profiles.admin, false),
		profiles.account_id IS NULL,
		COALESCE(profiles.color, 0)
	FROM (
		SELECT account_id, min(position) AS position
		FROM unnest($1::uuid[]) WITH ORDINALITY AS asked (account_id, position)
		GROUP BY account_id
	) asked
	LEFT JOIN profiles ON profiles.account_id = asked.account_id
	LEFT JOIN guest_codes ON guest_codes.account_id = asked.account_id
	WHERE (profiles.account_id IS NOT NULL OR guest_codes.account_id IS NOT NULL)
	  -- An account its player asked to delete reads as deleted until a sign-in brings it back.
	  AND NOT EXISTS (SELECT 1 FROM departures WHERE departures.account_id = asked.account_id AND departures.leaving)
	ORDER BY asked.position
`

func (q *PostgresQuery) Authors(ctx context.Context, accounts []cpsession.AccountID) (*playerv1.GetAuthorsResponse, error) {
	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.String())
	}

	rows, err := q.db.On(ctx).QueryContext(ctx, authors, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to read the authors: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &playerv1.GetAuthorsResponse{Authors: make([]*playerv1.Author, 0, len(accounts))}
	for rows.Next() {
		var (
			author = &playerv1.Author{}
			color  int32
		)
		if err := rows.Scan(&author.AccountId, &author.Name, &author.Admin, &author.Guest, &color); err != nil {
			return nil, fmt.Errorf("failed to scan an author: %w", err)
		}
		if author.Color, err = playerread.KeptColor(color); err != nil {
			return nil, fmt.Errorf("failed to read the color of %s: %w", author.GetAccountId(), err)
		}
		answer.Authors = append(answer.Authors, author)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the authors: %w", err)
	}
	return answer, nil
}
