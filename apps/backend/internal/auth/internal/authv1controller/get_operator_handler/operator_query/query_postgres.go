package operator_query

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const cookieName = "cp_admin"

var (
	ErrNoSession     = errors.New("this browser holds no admin session")
	ErrNotAnOperator = errors.New("not an operator")
)

func NewPostgresQuery(db cppg.Conn, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock}
}

type PostgresQuery struct {
	db    cppg.Conn
	clock cptime.Clock
}

// The flag is read again on every call: unsetting it ends a session already open.
const operator = `
	SELECT accounts.id, accounts.operator
	FROM admin_sessions
	JOIN accounts ON accounts.id = admin_sessions.account_id
	WHERE admin_sessions.token_hash = $1 AND admin_sessions.expires_at > $2
`

func (q *PostgresQuery) Operator(ctx context.Context, cookieHeader string) (*authv1.GetOperatorResponse, error) {
	hash, found := tokenHash(cookieHeader)
	if !found {
		return nil, fmt.Errorf("%w: the browser sent no %s cookie", ErrNoSession, cookieName)
	}

	var (
		account  uuid.UUID
		operates bool
	)
	// Postgres keeps microseconds and rounds what it is sent: truncated, "now < expires_at" stays exact.
	now := q.clock.Now().UTC().Truncate(time.Microsecond)
	err := q.db.On(ctx).QueryRowContext(ctx, operator, hash, now).Scan(&account, &operates)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: no live admin session for this cookie", ErrNoSession)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the caller: %w", err)
	}

	if !operates {
		return nil, fmt.Errorf("%w: account %s", ErrNotAnOperator, account)
	}

	return &authv1.GetOperatorResponse{AccountId: account.String()}, nil
}

func tokenHash(cookieHeader string) ([]byte, bool) {
	cookies, err := http.ParseCookie(cookieHeader)
	if err != nil {
		return nil, false
	}

	for _, cookie := range cookies {
		if cookie.Name == cookieName && cookie.Value != "" {
			sum := sha256.Sum256([]byte(cookie.Value))
			return sum[:], true
		}
	}
	return nil, false
}
