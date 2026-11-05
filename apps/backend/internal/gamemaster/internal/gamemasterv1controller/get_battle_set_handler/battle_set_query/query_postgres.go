package battle_set_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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

// RuleSet is the set a battle plays, by the id the timeline names it by: an entry of kind battle by its id, a season's
// finale by its round. A day, a plain entry or a season with no finale is no battle.
func (q *PostgresQuery) RuleSet(ctx context.Context, battleID string) (string, error) {
	named, err := moments.NamedOf(moments.RoundID(battleID))
	if err != nil || named.Day != 0 {
		return "", fmt.Errorf("%w: %q", ErrNoBattle, battleID)
	}

	var set string
	if named.Finale {
		err = q.db.On(ctx).QueryRowContext(ctx,
			`SELECT rule_set FROM season_finales WHERE season = $1`, int64(named.Season)).Scan(&set)
	} else {
		err = q.db.On(ctx).QueryRowContext(ctx,
			`SELECT rule_set FROM schedule_entries WHERE id = $1 AND kind = 'battle'`, named.Entry.String()).Scan(&set)
	}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", fmt.Errorf("%w: %q", ErrNoBattle, battleID)
	case err != nil:
		return "", fmt.Errorf("failed to read the rule set of the battle %q: %w", battleID, err)
	}
	return set, nil
}
