package caller_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/authread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewPostgresQuery(db cppg.Querier, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock}
}

type PostgresQuery struct {
	db    cppg.Querier
	clock cptime.Clock
}

const liveSession = `
	SELECT account_id
	FROM sessions
	WHERE ` + authread.LiveSession + `
`

func (q *PostgresQuery) Caller(ctx context.Context, cookieHeader string) (*authv1.GetCallerResponse, error) {
	hash, found := authread.TokenHash(cookieHeader)
	if !found {
		return &authv1.GetCallerResponse{}, nil
	}

	var account uuid.UUID
	err := q.db.QueryRowContext(ctx, liveSession, hash, authread.Now(q.clock)).Scan(&account)
	if errors.Is(err, sql.ErrNoRows) {
		return &authv1.GetCallerResponse{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read whose session the cookie holds: %w", err)
	}
	return &authv1.GetCallerResponse{AccountId: account.String()}, nil
}
