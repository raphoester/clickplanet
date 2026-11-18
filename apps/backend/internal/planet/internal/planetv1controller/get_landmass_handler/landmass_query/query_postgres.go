package landmass_query

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

// Who holds each tile is the map's, in memory: the ledger would read every change of every tile of the landmass.
type Holders interface {
	Holders(landmass clicks.LandmassID) []*planetv1.LandmassHolder
}

// Stages is the map that plays, on its board.
type Stages interface {
	Stage() clicks.Stage
}

func NewPostgresQuery(db cppg.Conn, holders Holders, stages Stages) *PostgresQuery {
	return &PostgresQuery{db: db, holders: holders, stages: stages}
}

type PostgresQuery struct {
	db      cppg.Conn
	holders Holders
	stages  Stages
}

// The last fortify on the map that plays, on its board ($2, $3), as each event names its map and its board: a
// landmass's id is its board's.
const lastFortify = `
	SELECT COALESCE(account::text, ''), country, taken_at FROM ledger_events
	WHERE kind = 'fortify' AND ((payload ->> 'landmass')::integer) = $1 AND map = $2 AND board = $3
	ORDER BY position DESC LIMIT 1
`

func (q *PostgresQuery) Landmass(ctx context.Context, landmass clicks.LandmassID) (*planetv1.GetLandmassResponse, error) {
	answer := &planetv1.GetLandmassResponse{Holders: q.holders.Holders(landmass)}
	playing := q.stages.Stage()

	var (
		fortification = &planetv1.LandmassFortification{}
		at            time.Time
	)
	err := q.db.On(ctx).QueryRowContext(ctx, lastFortify, int(landmass), string(playing.Map), playing.Board).
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
