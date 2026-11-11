package landmass_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Who holds each tile is the map's, in memory: the ledger would read every change of every tile of the landmass.
type Holders interface {
	Holders(landmass clicks.LandmassID) []*planetv1.LandmassHolder
}

type Arenas interface {
	Stretches(ctx context.Context) ([]arenas.Stretch, error)
}

func NewPostgresQuery(db cppg.Conn, holders Holders, arenas Arenas, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, holders: holders, arenas: arenas, clock: clock}
}

type PostgresQuery struct {
	db      cppg.Conn
	holders Holders
	arenas  Arenas
	clock   cptime.Clock
}

// The last fortify on the map that plays: after $2, and in none of the gaps ($3, $4).
const lastFortify = `
	SELECT COALESCE(account::text, ''), country, taken_at FROM ledger_events
	WHERE kind = 'fortify' AND ((payload ->> 'landmass')::integer) = $1 AND taken_at > $2
		AND NOT EXISTS (
			SELECT 1 FROM unnest($3::timestamptz[], $4::timestamptz[]) AS gap (after, before)
			WHERE taken_at > gap.after AND taken_at < gap.before
		)
	ORDER BY position DESC LIMIT 1
`

func (q *PostgresQuery) Landmass(ctx context.Context, landmass clicks.LandmassID) (*planetv1.GetLandmassResponse, error) {
	answer := &planetv1.GetLandmassResponse{Holders: q.holders.Holders(landmass)}
	stretches, err := q.arenas.Stretches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the arenas: %w", err)
	}
	playing := arenas.MapsOf(stretches).Playing()
	afters, befores := playing.Bounds(q.clock.Now())

	var (
		fortification = &planetv1.LandmassFortification{}
		at            time.Time
	)
	err = q.db.On(ctx).QueryRowContext(ctx, lastFortify, int(landmass), playing.After, pq.Array(afters), pq.Array(befores)).
		Scan(&fortification.AccountId, &fortification.CountryId, &at)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return answer, nil
	case err != nil:
		return nil, fmt.Errorf("failed to read the last fortify of landmass %d: %w", landmass, err)
	}
	fortification.FortifiedAtUnixMs = at.UnixMilli()
	answer.LastFortification = fortification

	return answer, nil
}
