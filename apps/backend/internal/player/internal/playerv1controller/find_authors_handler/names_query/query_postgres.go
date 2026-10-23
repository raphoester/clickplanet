package names_query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

// An account goes by its username, or by its guest code until it chooses one, as GetAuthors names it.
const named = `
	SELECT account_id, count(*) OVER ()
	FROM (
		SELECT account_id, name_folded AS folded FROM profiles
		UNION ALL
		SELECT account_id, 'guest_' || code FROM guest_codes
		WHERE NOT EXISTS (SELECT 1 FROM profiles WHERE profiles.account_id = guest_codes.account_id)
	) named
	WHERE strpos(folded, $1) > 0
	ORDER BY folded = $1 DESC, strpos(folded, $1) = 1 DESC, length(folded), folded
	LIMIT $2
`

func (q *PostgresQuery) Find(ctx context.Context, folded string, limit int) ([]players.AccountID, int, error) {
	rows, err := q.db.QueryContext(ctx, named, folded, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find the names: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var (
		accounts []players.AccountID
		total    int
	)
	for rows.Next() {
		var account uuid.UUID
		if err := rows.Scan(&account, &total); err != nil {
			return nil, 0, fmt.Errorf("failed to scan a name: %w", err)
		}
		accounts = append(accounts, players.AccountID(account))
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to find the names: %w", err)
	}

	return accounts, total, nil
}
