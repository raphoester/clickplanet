package accounts_query

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

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

const known = `
	SELECT id, created_at, EXISTS (SELECT 1 FROM identities WHERE identities.account_id = accounts.id)
	FROM accounts WHERE id = ANY($1::uuid[])
	ORDER BY id
`

func (q *PostgresQuery) Accounts(ctx context.Context, asked []cpsession.AccountID) (*authv1.GetAccountsResponse, error) {
	ids := make([]string, len(asked))
	for i, account := range asked {
		ids[i] = account.String()
	}

	rows, err := q.db.QueryContext(ctx, known, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to select the accounts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &authv1.GetAccountsResponse{Accounts: make([]*authv1.Account, 0, len(asked))}
	for rows.Next() {
		var (
			id        uuid.UUID
			createdAt time.Time
			linked    bool
		)
		if err := rows.Scan(&id, &createdAt, &linked); err != nil {
			return nil, fmt.Errorf("failed to read an account: %w", err)
		}
		answer.Accounts = append(answer.Accounts, &authv1.Account{
			AccountId: id.String(), Linked: linked, CreatedAtUnixMs: createdAt.UnixMilli(),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the accounts: %w", err)
	}
	return answer, nil
}
