package allegiance_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/allegiancemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

func (q *PostgresQuery) Allegiance(ctx context.Context, account allegiances.Account) (*planetv1.GetAllegianceResponse, error) {
	if account == allegiances.NoAccount {
		return &planetv1.GetAllegianceResponse{}, nil
	}

	var (
		flag      string
		changedAt sql.NullTime
	)
	err := q.db.QueryRowContext(ctx, `SELECT flag, changed_at FROM allegiances WHERE account = $1`, string(account)).
		Scan(&flag, &changedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return &planetv1.GetAllegianceResponse{}, nil
	case err != nil:
		return nil, fmt.Errorf("failed to read the allegiance of %s: %w", account, err)
	}

	return &planetv1.GetAllegianceResponse{
		Allegiance: allegiancemessage.Encode(flag, allegiances.ChangedAt(changedAt.Time).ChangeableAt()),
	}, nil
}
