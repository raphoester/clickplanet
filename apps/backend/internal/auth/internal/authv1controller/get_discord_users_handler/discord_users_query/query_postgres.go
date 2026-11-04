package discord_users_query

import (
	"context"
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

// One row per account, so a page that ends on an account never splits it: an account holds one Discord user at most,
// and DISTINCT ON keeps that true whatever the table holds. The subject of a Discord identity is the Discord user id.
const discordUsers = `
	SELECT DISTINCT ON (identities.account_id) identities.account_id, identities.subject
	FROM identities JOIN accounts ON accounts.id = identities.account_id
	WHERE identities.provider = 'discord' AND accounts.deletion_requested_at IS NULL AND identities.account_id > $1
	ORDER BY identities.account_id, identities.linked_at
	LIMIT $2
`

func (q *PostgresQuery) DiscordUsers(
	ctx context.Context, after cpsession.AccountID, limit int,
) (*authv1.GetDiscordUsersResponse, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, discordUsers, uuid.UUID(after), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to select the Discord users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &authv1.GetDiscordUsersResponse{}
	for rows.Next() {
		var (
			account uuid.UUID
			subject string
		)
		if err := rows.Scan(&account, &subject); err != nil {
			return nil, fmt.Errorf("failed to read a Discord user: %w", err)
		}
		answer.Users = append(answer.Users, &authv1.DiscordUser{AccountId: account.String(), DiscordUserId: subject})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the Discord users: %w", err)
	}
	return answer, nil
}
