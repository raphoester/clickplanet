package player_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

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

// How many of the names its player left the card shows.
const PastNamesShown = 5

// Each name once, at the last time it was left, and never the one held now: a player that came back to a name holds
// it, and two changes saved at once may have kept one name twice.
const pastNames = `
	SELECT name, left_at FROM (
		SELECT DISTINCT ON (past_names.name_folded) past_names.name, past_names.left_at, past_names.id
		FROM past_names
		JOIN profiles ON profiles.account_id = past_names.account_id
		WHERE past_names.account_id = $1 AND past_names.name_folded <> profiles.name_folded
		ORDER BY past_names.name_folded, past_names.left_at DESC, past_names.id DESC
	) latest
	ORDER BY left_at DESC, id DESC
	LIMIT $2
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

	var (
		createdAt time.Time
		left      []*playerv1.PastName
	)
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		var err error
		createdAt, err = q.accounts.CreatedAt(groupCtx, account)
		if err != nil {
			return fmt.Errorf("failed to ask when the account was made: %w", err)
		}
		return nil
	})
	group.Go(func() error {
		var err error
		left, err = q.pastNames(groupCtx, account)
		return err
	})
	if err := group.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // each part names what failed.
	}

	if !createdAt.IsZero() {
		answer.CreatedAtUnixMs = createdAt.UnixMilli()
	}
	return &playerv1.GetPlayerResponse{Player: answer, PastNames: left}, nil
}

func (q *PostgresQuery) pastNames(ctx context.Context, account cpsession.AccountID) ([]*playerv1.PastName, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, pastNames, uuid.UUID(account), PastNamesShown)
	if err != nil {
		return nil, fmt.Errorf("failed to read the names the player left: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var left []*playerv1.PastName
	for rows.Next() {
		var (
			name   string
			leftAt time.Time
		)
		if err := rows.Scan(&name, &leftAt); err != nil {
			return nil, fmt.Errorf("failed to read a name the player left: %w", err)
		}
		left = append(left, &playerv1.PastName{Name: name, LeftAtUnixMs: leftAt.UnixMilli()})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the names the player left: %w", err)
	}
	return left, nil
}
