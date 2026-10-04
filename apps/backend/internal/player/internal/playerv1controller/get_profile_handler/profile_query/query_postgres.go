package profile_query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

const profile = `
	SELECT COALESCE(profiles.name, ''), COALESCE(profiles.color, 0)
	FROM (SELECT $1::uuid AS account_id) asked
	LEFT JOIN profiles ON profiles.account_id = asked.account_id
`

func (q *PostgresQuery) Profile(ctx context.Context, account players.AccountID) (*playerv1.GetProfileResponse, error) {
	var (
		name  string
		color int32
	)
	if err := q.db.QueryRowContext(ctx, profile, uuid.UUID(account)).Scan(&name, &color); err != nil {
		return nil, fmt.Errorf("failed to read the profile: %w", err)
	}

	kept, err := playermessage.KeptColor(color)
	if err != nil {
		return nil, fmt.Errorf("failed to read the profile's color: %w", err)
	}

	return &playerv1.GetProfileResponse{
		Profile: &playerv1.Profile{AccountId: account.String(), Name: name},
		Color:   kept,
	}, nil
}
