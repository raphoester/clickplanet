package province_query

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

// Board is the board in play: its provinces file names the provinces a fortify's id is a row of.
type Board interface {
	Provinces() *clicks.Provinces
}

func NewPostgresQuery(db cppg.Conn, stages Stages, board Board) *PostgresQuery {
	return &PostgresQuery{db: db, stages: stages, board: board}
}

type PostgresQuery struct {
	db     cppg.Conn
	stages Stages
	board  Board
}

// The last fortify on the map that plays, on its board ($3, $4), of the provinces file in play ($2), as each event names
// them: a province's id is a row of its file, and a fortify made before provinces names none.
const lastFortify = `
	SELECT COALESCE(account::text, ''), country, taken_at FROM ledger_events
	WHERE kind = 'fortify' AND ((payload ->> 'province')::integer) = $1 AND payload ->> 'provinces' = $2
		AND map = $3 AND board = $4
	ORDER BY position DESC LIMIT 1
`

func (q *PostgresQuery) Province(ctx context.Context, province clicks.ProvinceID) (*planetv1.GetProvinceResponse, error) {
	answer := &planetv1.GetProvinceResponse{}
	playing := q.stages.Stage()

	var (
		fortification = &planetv1.ProvinceFortification{}
		at            time.Time
	)
	err := q.db.On(ctx).QueryRowContext(ctx, lastFortify, int(province), q.board.Provinces().Asset(),
		string(playing.Map), playing.Board).Scan(&fortification.AccountId, &fortification.CountryId, &at)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return answer, nil
	case err != nil:
		return nil, fmt.Errorf("failed to read the last fortify of province %d: %w", province, err)
	}
	fortification.FortifiedAtUnixMs = at.UnixMilli()
	answer.LastFortification = fortification

	return answer, nil
}
