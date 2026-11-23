package email_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

const email = `
	SELECT email FROM identities
	WHERE account_id = $1 AND email IS NOT NULL
	ORDER BY linked_at, provider::text
	LIMIT 1
`

func (q *PostgresQuery) Email(ctx context.Context, asked cpsession.AccountID) (*authv1.GetEmailResponse, error) {
	var held string
	err := q.db.On(ctx).QueryRowContext(ctx, email, uuid.UUID(asked)).Scan(&held)
	if errors.Is(err, sql.ErrNoRows) {
		return &authv1.GetEmailResponse{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the address of the account: %w", err)
	}
	return &authv1.GetEmailResponse{Email: &held}, nil
}
