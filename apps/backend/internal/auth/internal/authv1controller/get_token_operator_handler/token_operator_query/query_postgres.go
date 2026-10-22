package token_operator_query

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	ErrNoToken       = errors.New("no live access token")
	ErrNotAnOperator = errors.New("not an operator")
)

func NewPostgresQuery(db cppg.Querier, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock}
}

type PostgresQuery struct {
	db    cppg.Querier
	clock cptime.Clock
}

// The flag is read again on every call, as for an admin session: unsetting it ends every connection of the account.
// A refresh token is never one: it only buys the next pair, at the token endpoint.
const operator = `
	SELECT admin_tokens.client, accounts.id, accounts.operator
	FROM admin_tokens
	JOIN accounts ON accounts.id = admin_tokens.account_id
	WHERE admin_tokens.token_hash = $1 AND admin_tokens.kind = 'access' AND admin_tokens.expires_at > $2
`

// The hash is the SHA-256 of the token as it was sent, the rule admintokens.Secret keeps it by.
func (q *PostgresQuery) Operator(ctx context.Context, token string) (*authv1.GetTokenOperatorResponse, error) {
	if token == "" {
		return nil, fmt.Errorf("%w: the caller sent no token", ErrNoToken)
	}

	var (
		client   string
		account  uuid.UUID
		operates bool
		hash     = sha256.Sum256([]byte(token))
		now      = q.clock.Now().UTC().Truncate(time.Microsecond)
	)
	err := q.db.QueryRowContext(ctx, operator, hash[:], now).Scan(&client, &account, &operates)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: nobody holds this token, or it lapsed", ErrNoToken)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the token: %w", err)
	}

	if !operates {
		return nil, fmt.Errorf("%w: account %s", ErrNotAnOperator, account)
	}

	return &authv1.GetTokenOperatorResponse{AccountId: account.String(), Client: client}, nil
}
