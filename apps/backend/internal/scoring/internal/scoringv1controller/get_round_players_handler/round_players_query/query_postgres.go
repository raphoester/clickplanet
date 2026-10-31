package round_players_query

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var ErrNoRound = errors.New("no round has this id")

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

func (q *PostgresQuery) Players(ctx context.Context, id books.RoundID) ([]*scoringv1.RoundPlayer, error) {
	var opened bool
	if err := q.db.On(ctx).QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM rounds WHERE round_id = $1)
	`, string(id)).Scan(&opened); err != nil {
		return nil, fmt.Errorf("failed to read the round: %w", err)
	}
	if !opened {
		return nil, fmt.Errorf("%w: %q", ErrNoRound, id)
	}

	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT DISTINCT ON (account_id) account_id, country, tiles FROM round_players
		WHERE round_id = $1
		ORDER BY account_id, tiles DESC, country
	`, string(id))
	if err != nil {
		return nil, fmt.Errorf("failed to read the players of the round: %w", err)
	}
	defer func() { _ = rows.Close() }()

	players := []*scoringv1.RoundPlayer{}
	for rows.Next() {
		var (
			account uuid.UUID
			tiles   int64
			player  = &scoringv1.RoundPlayer{}
		)
		if err := rows.Scan(&account, &player.CountryId, &tiles); err != nil {
			return nil, fmt.Errorf("failed to read a player of the round: %w", err)
		}
		player.AccountId, player.Tiles = account.String(), uint64(tiles) //nolint:gosec // CHECK (tiles > 0).
		players = append(players, player)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the players of the round: %w", err)
	}
	return players, nil
}
