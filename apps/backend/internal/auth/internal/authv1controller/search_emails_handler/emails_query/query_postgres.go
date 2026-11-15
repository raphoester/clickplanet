package emails_query

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

// strpos and not LIKE, so a "_" or a "%" typed is matched as itself. The order breaks every tie, so a page never
// repeats or skips a row of the one before.
const holders = `
	SELECT account_id, provider::text, email, linked_at
	FROM identities
	WHERE email_verified AND strpos(lower(email), lower($1)) > 0
	ORDER BY lower(email), linked_at, provider::text, account_id
	LIMIT $2 OFFSET $3
`

func (q *PostgresQuery) Holders(
	ctx context.Context, text string, limit int, offset int,
) (*authv1.SearchEmailsResponse, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, holders, text, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to select the holders of the addresses: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &authv1.SearchEmailsResponse{}
	for rows.Next() {
		var (
			account  uuid.UUID
			provider string
			email    string
			linkedAt time.Time
		)
		if err := rows.Scan(&account, &provider, &email, &linkedAt); err != nil {
			return nil, fmt.Errorf("failed to read a holder of an address: %w", err)
		}
		answer.Holders = append(answer.Holders, &authv1.EmailHolder{
			AccountId: account.String(), Email: email, Provider: provider, LinkedAt: timestamppb.New(linkedAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the holders of the addresses: %w", err)
	}
	return answer, nil
}
