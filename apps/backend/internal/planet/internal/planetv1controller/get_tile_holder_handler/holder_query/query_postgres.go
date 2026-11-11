package holder_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Arenas interface {
	Stretches(ctx context.Context) ([]arenas.Stretch, error)
}

func NewPostgresQuery(db cppg.Conn, arenas Arenas, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, arenas: arenas, clock: clock}
}

type PostgresQuery struct {
	db     cppg.Conn
	arenas Arenas
	clock  cptime.Clock
}

// The last of the changes ledger.Taking replays for the tile on the map that plays: a take, a tile a spread or an
// enclose took, or a bomb's clear, after $2 and in none of the gaps ($3, $4).
const lastChange = `
	WITH gap AS (SELECT * FROM unnest($3::timestamptz[], $4::timestamptz[]) AS gap (after, before))
	SELECT COALESCE(account::text, ''), country, taken_at FROM (
		(SELECT position, account, country, taken_at FROM ledger_events e
		WHERE kind = 'take' AND tile = $1 AND NOT forgotten AND taken_at > $2
			AND NOT EXISTS (SELECT 1 FROM gap WHERE e.taken_at > gap.after AND e.taken_at < gap.before)
		ORDER BY position DESC LIMIT 1)
		UNION ALL
		(SELECT position, account, country, taken_at FROM ledger_events e
		WHERE kind IN ('spread', 'enclose') AND NOT forgotten AND taken_at > $2
			AND payload @> jsonb_build_object('taken', jsonb_build_array(jsonb_build_object('tile', $1::integer)))
			AND NOT EXISTS (SELECT 1 FROM gap WHERE e.taken_at > gap.after AND e.taken_at < gap.before)
		ORDER BY position DESC LIMIT 1)
		UNION ALL
		(SELECT position, account, '', taken_at FROM ledger_events e
		WHERE kind = 'bomb' AND NOT forgotten AND taken_at > $2
			AND payload @> jsonb_build_object('cleared', jsonb_build_array(jsonb_build_object('tile', $1::integer)))
			AND NOT EXISTS (SELECT 1 FROM gap WHERE e.taken_at > gap.after AND e.taken_at < gap.before)
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
	stretches, err := q.arenas.Stretches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the arenas: %w", err)
	}
	playing := arenas.MapsOf(stretches).Playing()
	afters, befores := playing.Bounds(q.clock.Now())
	err = q.db.On(ctx).QueryRowContext(ctx, lastChange, tile, playing.After, pq.Array(afters), pq.Array(befores)).Scan(&holder.AccountId, &holder.CountryId, &takenAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return &planetv1.GetTileHolderResponse{}, nil
	case err != nil:
		return nil, fmt.Errorf("failed to read who holds tile %d: %w", tile, err)
	}
	holder.TakenAtUnixMs = takenAt.UnixMilli()
	return &planetv1.GetTileHolderResponse{Holder: holder}, nil
}
