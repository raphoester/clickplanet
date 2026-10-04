package me_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/authread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var providers = map[string]authv1.Provider{
	"google":  authv1.Provider_PROVIDER_GOOGLE,
	"discord": authv1.Provider_PROVIDER_DISCORD,
	"email":   authv1.Provider_PROVIDER_EMAIL,
}

var (
	ErrNoAccount       = errors.New("this browser has no account")
	ErrUnknownProvider = errors.New("a kept provider the proto does not name")
)

func NewPostgresQuery(db cppg.Querier, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock}
}

type PostgresQuery struct {
	db    cppg.Querier
	clock cptime.Clock
}

const me = `
	SELECT accounts.id,
	       COALESCE(array_agg(identities.provider ORDER BY identities.linked_at, identities.provider)
	                FILTER (WHERE identities.provider IS NOT NULL), '{}')
	FROM sessions
	JOIN accounts ON accounts.id = sessions.account_id
	LEFT JOIN identities ON identities.account_id = accounts.id
	WHERE ` + authread.LiveSession + `
	GROUP BY accounts.id
`

func (q *PostgresQuery) Me(ctx context.Context, cookieHeader string) (*authv1.GetMeResponse, error) {
	hash, found := authread.TokenHash(cookieHeader)
	if !found {
		return nil, fmt.Errorf("%w: the browser sent no %s cookie", ErrNoAccount, authread.SessionCookie)
	}

	var (
		account uuid.UUID
		names   []string
	)
	err := q.db.QueryRowContext(ctx, me, hash, authread.Now(q.clock)).Scan(&account, pq.Array(&names))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: no live session for this cookie", ErrNoAccount)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the caller: %w", err)
	}

	answer := &authv1.GetMeResponse{AccountId: account.String(), Kind: authv1.AccountKind_ACCOUNT_KIND_GUEST}
	for _, name := range names {
		provider, known := providers[name]
		if !known {
			return nil, fmt.Errorf("%w: %q", ErrUnknownProvider, name)
		}
		answer.Providers = append(answer.Providers, provider)
	}
	if len(answer.Providers) > 0 {
		answer.Kind = authv1.AccountKind_ACCOUNT_KIND_LINKED
	}
	return answer, nil
}
