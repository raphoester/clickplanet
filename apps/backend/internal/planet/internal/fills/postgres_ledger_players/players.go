package postgres_ledger_players

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Querier) *Players {
	return &Players{db: db}
}

type Players struct {
	db cppg.Querier
}

// One index probe per account, then one row of it to tell it played before: never a scan of every event.
const players = `
	WITH RECURSIVE accounts AS (
		(SELECT account FROM ledger_events WHERE account IS NOT NULL AND account > $1 ORDER BY account LIMIT 1)
		UNION ALL
		SELECT (
			SELECT e.account FROM ledger_events e
			WHERE e.account IS NOT NULL AND e.account > a.account ORDER BY e.account LIMIT 1
		)
		FROM accounts a WHERE a.account IS NOT NULL
	)
	SELECT account FROM accounts a
	WHERE account IS NOT NULL
		AND EXISTS (SELECT 1 FROM ledger_events e WHERE e.account = a.account AND e.taken_at < $2)
	LIMIT $3`

func (p *Players) Players(ctx context.Context, page fills.Page) ([]bonuses.Holder, error) {
	after := uuid.Nil
	if page.After != "" {
		id, err := uuid.Parse(string(page.After))
		if err != nil {
			return nil, fmt.Errorf("a fill reached %q, which is not an account: %w", page.After, err)
		}
		after = id
	}

	rows, err := p.db.QueryContext(ctx, players, after, page.Before.UTC(), page.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to read who played before %s: %w", page.Before, err)
	}
	defer func() { _ = rows.Close() }()

	var holders []bonuses.Holder
	for rows.Next() {
		var account uuid.UUID
		if err := rows.Scan(&account); err != nil {
			return nil, fmt.Errorf("failed to read a player: %w", err)
		}
		holders = append(holders, bonuses.Holder(account.String()))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read who played before %s: %w", page.Before, err)
	}

	return holders, nil
}
