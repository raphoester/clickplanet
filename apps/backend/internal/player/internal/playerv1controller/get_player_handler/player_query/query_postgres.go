package player_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Accounts interface {
	CreatedAt(ctx context.Context, account cpsession.AccountID) (time.Time, error)
}

var ErrNoPlayer = errors.New("no such player")

func NewPostgresQuery(db cppg.Conn, accounts Accounts) *PostgresQuery {
	return &PostgresQuery{db: db, accounts: accounts}
}

type PostgresQuery struct {
	db       cppg.Conn
	accounts Accounts
}

const player = `
	SELECT
		profiles.account_id IS NULL,
		COALESCE(profiles.name, 'guest_' || guest_codes.code),
		COALESCE(profiles.admin, false),
		COALESCE(profiles.color, 0)
	FROM (SELECT $1::uuid AS account_id) asked
	LEFT JOIN profiles ON profiles.account_id = asked.account_id
	LEFT JOIN guest_codes ON guest_codes.account_id = asked.account_id
	WHERE (profiles.account_id IS NOT NULL OR guest_codes.account_id IS NOT NULL)
	  -- An account its player asked to delete reads as deleted until a sign-in brings it back.
	  AND NOT EXISTS (SELECT 1 FROM departures WHERE departures.account_id = asked.account_id AND departures.leaving)
`

func (q *PostgresQuery) Player(ctx context.Context, account cpsession.AccountID) (*playerv1.GetPlayerResponse, error) {
	var (
		guest  bool
		color  int32
		answer = &playerv1.Player{AccountId: account.String()}
	)
	err := q.db.On(ctx).QueryRowContext(ctx, player, uuid.UUID(account)).Scan(&guest, &answer.Name, &answer.Admin, &color)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoPlayer
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the player: %w", err)
	}
	if guest {
		return &playerv1.GetPlayerResponse{Player: &playerv1.Player{AccountId: answer.GetAccountId(), Name: answer.GetName(), Guest: true}}, nil
	}

	if answer.Color, err = playerread.KeptColor(color); err != nil {
		return nil, fmt.Errorf("failed to read the player's color: %w", err)
	}

	createdAt, err := q.accounts.CreatedAt(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to ask when the account was made: %w", err)
	}
	if !createdAt.IsZero() {
		answer.CreatedAtUnixMs = createdAt.UnixMilli()
	}
	return &playerv1.GetPlayerResponse{Player: answer}, nil
}
