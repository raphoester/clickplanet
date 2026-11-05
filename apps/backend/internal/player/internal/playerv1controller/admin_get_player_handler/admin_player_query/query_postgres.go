package admin_player_query

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Accounts interface {
	Account(ctx context.Context, account cpsession.AccountID) (players.Account, error)
}

var ErrNoPlayer = errors.New("the game keeps nothing about this account")

func NewPostgresQuery(db cppg.Conn, accounts Accounts) *PostgresQuery {
	return &PostgresQuery{db: db, accounts: accounts}
}

type PostgresQuery struct {
	db       cppg.Conn
	accounts Accounts
}

const player = `
	SELECT
		profiles.account_id IS NOT NULL OR guest_codes.account_id IS NOT NULL,
		COALESCE(profiles.name, 'guest_' || guest_codes.code, ''),
		profiles.account_id IS NULL,
		COALESCE(profiles.admin, false),
		COALESCE(profiles.color, 0)
	FROM (SELECT $1::uuid AS account_id) asked
	LEFT JOIN profiles ON profiles.account_id = asked.account_id
	LEFT JOIN guest_codes ON guest_codes.account_id = asked.account_id
`

// An account never shown has no name and no guest code: it is still one while auth knows it.
func (q *PostgresQuery) Player(ctx context.Context, account cpsession.AccountID) (*playerv1.AdminGetPlayerResponse, error) {
	var (
		shown  bool
		color  int32
		answer = &playerv1.AdminGetPlayerResponse{}
	)
	if err := q.db.On(ctx).QueryRowContext(ctx, player, uuid.UUID(account)).Scan(
		&shown, &answer.Name, &answer.Guest, &answer.Admin, &color,
	); err != nil {
		return nil, fmt.Errorf("failed to read the player: %w", err)
	}

	var err error
	if answer.Color, err = playerread.KeptColor(color); err != nil {
		return nil, fmt.Errorf("failed to read the player's color: %w", err)
	}

	kept, err := q.accounts.Account(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to ask auth about the account: %w", err)
	}
	if !shown && !kept.Linked() && kept.CreatedAt().IsZero() {
		return nil, ErrNoPlayer
	}
	answer.Linked = kept.Linked()
	if !kept.CreatedAt().IsZero() {
		answer.CreatedAtUnixMs = kept.CreatedAt().UnixMilli()
	}
	return answer, nil
}
