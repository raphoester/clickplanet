package holder_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

// The last of the changes ledger.Taking replays for the tile: a take, a tile a spread or an enclose took, or a bomb's clear.
const lastChange = `
	SELECT COALESCE(account::text, ''), country, taken_at FROM (
		(SELECT position, account, country, taken_at FROM ledger_events
		WHERE kind = 'take' AND tile = $1
		ORDER BY position DESC LIMIT 1)
		UNION ALL
		(SELECT position, account, country, taken_at FROM ledger_events
		WHERE kind IN ('spread', 'enclose')
			AND payload @> jsonb_build_object('taken', jsonb_build_array(jsonb_build_object('tile', $1::integer)))
		ORDER BY position DESC LIMIT 1)
		UNION ALL
		(SELECT position, account, '', taken_at FROM ledger_events
		WHERE kind = 'bomb'
			AND payload @> jsonb_build_object('cleared', jsonb_build_array(jsonb_build_object('tile', $1::integer)))
		ORDER BY position DESC LIMIT 1)
		ORDER BY position DESC LIMIT 1
	) last
	WHERE country <> ''
`

func (q *PostgresQuery) TileHolder(ctx context.Context, tile uint32) (*planetv1.GetTileHolderResponse, error) {
	var (
		holder  = &planetv1.TileHolder{}
		takenAt time.Time
	)
	err := q.db.QueryRowContext(ctx, lastChange, tile).Scan(&holder.AccountId, &holder.CountryId, &takenAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return &planetv1.GetTileHolderResponse{}, nil
	case err != nil:
		return nil, fmt.Errorf("failed to read who holds tile %d: %w", tile, err)
	}
	holder.TakenAtUnixMs = takenAt.UnixMilli()
	return &planetv1.GetTileHolderResponse{Holder: holder}, nil
}
