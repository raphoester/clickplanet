package matches_query

import (
	"context"
	"fmt"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

// An account goes by its username, or by its guest code until it chooses one, as GetAuthors names it.
const matches = `
	SELECT account_id::text, name, guest, count(*) OVER ()
	FROM (
		SELECT account_id, name, name_folded AS folded, false AS guest FROM profiles
		UNION ALL
		SELECT account_id, 'guest_' || code, 'guest_' || code, true FROM guest_codes
		WHERE NOT EXISTS (SELECT 1 FROM profiles WHERE profiles.account_id = guest_codes.account_id)
	) named
	WHERE strpos(folded, $1) > 0
	ORDER BY folded = $1 DESC, strpos(folded, $1) = 1 DESC, length(folded), folded
	LIMIT $2
`

func (q *PostgresQuery) Search(ctx context.Context, folded string, limit int) (*playerv1.SearchPlayersResponse, error) {
	rows, err := q.db.QueryContext(ctx, matches, folded, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search the players: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &playerv1.SearchPlayersResponse{}
	for rows.Next() {
		match := &playerv1.PlayerMatch{}
		if err := rows.Scan(&match.AccountId, &match.Name, &match.Guest, &answer.Total); err != nil {
			return nil, fmt.Errorf("failed to scan a player: %w", err)
		}
		answer.Players = append(answer.Players, match)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to search the players: %w", err)
	}

	return answer, nil
}
