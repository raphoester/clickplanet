package link_query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Conn) PostgresQuery {
	return PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

func (q PostgresQuery) Linked(ctx context.Context, account links.AccountID) (bool, error) {
	var linked bool
	if err := q.db.On(ctx).QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM links WHERE account_id = $1)
	`, uuid.UUID(account)).Scan(&linked); err != nil {
		return false, fmt.Errorf("failed to read whether the account linked Discord: %w", err)
	}
	return linked, nil
}
