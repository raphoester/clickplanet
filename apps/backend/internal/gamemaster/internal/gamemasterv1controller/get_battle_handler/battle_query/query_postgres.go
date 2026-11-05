package battle_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var ErrNoBattle = errors.New("no battle has this id")

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

// Battle is what a battle plays, by the id the timeline names it by: an entry of kind battle by its id, a season's
// finale by its round. A day, a plain entry or a season with no finale is no battle.
func (q *PostgresQuery) Battle(ctx context.Context, battleID string) (*gamemasterv1.GetBattleResponse, error) {
	named, err := moments.NamedOf(moments.RoundID(battleID))
	if err != nil || named.Day != 0 {
		return nil, fmt.Errorf("%w: %q", ErrNoBattle, battleID)
	}

	var (
		set   string
		times int64
	)
	if named.Finale {
		err = q.db.On(ctx).QueryRowContext(ctx,
			`SELECT rule_set, times FROM season_finales WHERE season = $1`, int64(named.Season)).Scan(&set, &times)
	} else {
		err = q.db.On(ctx).QueryRowContext(ctx,
			`SELECT rule_set, times FROM schedule_entries WHERE id = $1 AND kind = 'battle'`, named.Entry.String()).
			Scan(&set, &times)
	}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, fmt.Errorf("%w: %q", ErrNoBattle, battleID)
	case err != nil:
		return nil, fmt.Errorf("failed to read the battle %q: %w", battleID, err)
	}
	return &gamemasterv1.GetBattleResponse{RuleSet: set, Times: uint32(times)}, nil //nolint:gosec // 1 to 10, checked by the table.
}
