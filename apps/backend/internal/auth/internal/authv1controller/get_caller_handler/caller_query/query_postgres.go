package caller_query

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const cookieName = "cp_sid"

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
	WHERE token_hash = $1 AND expires_at > $2
`

func (q *PostgresQuery) Caller(ctx context.Context, cookieHeader string) (*authv1.GetCallerResponse, error) {
	token, found := tokenOf(cookieHeader)
	if !found {
		return &authv1.GetCallerResponse{}, nil
	}

	hash := sha256.Sum256([]byte(token))
	var account uuid.UUID
	err := q.db.QueryRowContext(ctx, liveSession, hash[:], q.clock.Now()).Scan(&account)
	if errors.Is(err, sql.ErrNoRows) {
		return &authv1.GetCallerResponse{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read whose session the cookie holds: %w", err)
	}
	return &authv1.GetCallerResponse{AccountId: account.String()}, nil
}

func tokenOf(cookieHeader string) (string, bool) {
	cookies, err := http.ParseCookie(cookieHeader)
	if err != nil {
		return "", false
	}
	for _, cookie := range cookies {
		if cookie.Name == cookieName && cookie.Value != "" {
			return cookie.Value, true
		}
	}
	return "", false
}
