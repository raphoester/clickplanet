package fronts_query

import (
	"context"

	"github.com/google/uuid"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

func (q *PostgresQuery) Fronts(ctx context.Context, account cpsession.AccountID) (*playerv1.GetFrontsResponse, error) {
	playsFor, playsAgainst, err := playerread.Fronts(ctx, q.db, uuid.UUID(account))
	if err != nil {
		return nil, err //nolint:wrapcheck // it names what failed.
	}
	return &playerv1.GetFrontsResponse{PlaysFor: playsFor, PlaysAgainst: playsAgainst}, nil
}
