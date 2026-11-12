package holder_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var ErrNoHolder = errors.New("no player holds this name")

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

const holder = `
	SELECT account_id::text, name, admin, color
	FROM profiles
	WHERE name_folded = $1
	  -- An account its player asked to delete reads as deleted until a sign-in brings it back.
	  AND NOT EXISTS (SELECT 1 FROM departures WHERE departures.account_id = profiles.account_id AND departures.leaving)
`

func (q *PostgresQuery) Holder(ctx context.Context, folded string) (*playerv1.GetPlayerByNameResponse, error) {
	var (
		player = &playerv1.Player{}
		color  int32
	)
	err := q.db.On(ctx).QueryRowContext(ctx, holder, folded).Scan(&player.AccountId, &player.Name, &player.Admin, &color)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoHolder
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read who holds the name: %w", err)
	}
	if player.Color, err = playerread.KeptColor(color); err != nil {
		return nil, fmt.Errorf("failed to read the color of %s: %w", player.GetAccountId(), err)
	}
	return &playerv1.GetPlayerByNameResponse{Player: player}, nil
}
