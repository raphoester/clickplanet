package account_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

const account = `
	SELECT created_at, EXISTS (SELECT 1 FROM identities WHERE identities.account_id = accounts.id)
	FROM accounts WHERE id = $1
`

func (q *PostgresQuery) Account(ctx context.Context, asked cpsession.AccountID) (*authv1.GetAccountResponse, error) {
	var (
		createdAt time.Time
		linked    bool
	)
	err := q.db.QueryRowContext(ctx, account, uuid.UUID(asked)).Scan(&createdAt, &linked)
	if errors.Is(err, sql.ErrNoRows) {
		return &authv1.GetAccountResponse{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the account: %w", err)
	}
	return &authv1.GetAccountResponse{Linked: linked, CreatedAtUnixMs: createdAt.UnixMilli()}, nil
}
