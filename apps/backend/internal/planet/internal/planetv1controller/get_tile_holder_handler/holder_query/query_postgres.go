package holder_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// Stages is the map that plays, on its board.
type Stages interface {
	Stage() clicks.Stage
}

func NewPostgresQuery(db cppg.Conn, stages Stages) *PostgresQuery {
	return &PostgresQuery{db: db, stages: stages}
}

type PostgresQuery struct {
	db     cppg.Conn
	stages Stages
}

// The last of the changes ledger.Taking replays for the tile on the map that plays, on its board ($2, $3): a take, a
// tile a spread, an enclose, a virus or a mine took, or a bomb's clear, as each event names its map and its board. Each
// kind of payload reads its own index.
const lastChange = `
	SELECT COALESCE(account::text, ''), country, taken_at FROM (
		(SELECT position, account, country, taken_at FROM ledger_events e
		WHERE kind = 'take' AND tile = $1 AND NOT forgotten AND map = $2 AND board = $3
		ORDER BY position DESC LIMIT 1)
		UNION ALL
		(SELECT position, account, country, taken_at FROM ledger_events e
		WHERE kind IN ('spread', 'enclose') AND NOT forgotten AND map = $2 AND board = $3
			AND payload @> jsonb_build_object('taken', jsonb_build_array(jsonb_build_object('tile', $1::integer)))
		ORDER BY position DESC LIMIT 1)
		UNION ALL
		(SELECT position, account, country, taken_at FROM ledger_events e
		WHERE kind = 'virus' AND NOT forgotten AND map = $2 AND board = $3
			AND payload @> jsonb_build_object('taken', jsonb_build_array(jsonb_build_object('tile', $1::integer)))
		ORDER BY position DESC LIMIT 1)
		UNION ALL
		(SELECT position, account, country, taken_at FROM ledger_events e
		WHERE kind = 'mine' AND NOT forgotten AND map = $2 AND board = $3
			AND payload @> jsonb_build_object('taken', jsonb_build_array(jsonb_build_object('tile', $1::integer)))
		ORDER BY position DESC LIMIT 1)
		UNION ALL
		(SELECT position, account, '', taken_at FROM ledger_events e
		WHERE kind = 'bomb' AND NOT forgotten AND map = $2 AND board = $3
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
	playing := q.stages.Stage()
	err := q.db.On(ctx).QueryRowContext(ctx, lastChange, tile, string(playing.Map), playing.Board).
		Scan(&holder.AccountId, &holder.CountryId, &takenAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return &planetv1.GetTileHolderResponse{}, nil
	case err != nil:
		return nil, fmt.Errorf("failed to read who holds tile %d: %w", tile, err)
	}
	holder.TakenAtUnixMs = takenAt.UnixMilli()
	return &planetv1.GetTileHolderResponse{Holder: holder}, nil
}
