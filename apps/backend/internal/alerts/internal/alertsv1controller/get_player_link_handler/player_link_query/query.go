package player_link_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	alertsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/alerts/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var ErrNoLink = errors.New("this account has no Discord link")

func NewPostgresQuery(db cppg.Conn) PostgresQuery {
	return PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

func (q PostgresQuery) LinkOf(ctx context.Context, account links.AccountID) (*alertsv1.DiscordLink, error) {
	var (
		user     string
		flag     sql.NullString
		linkedAt time.Time
	)
	err := q.db.On(ctx).QueryRowContext(ctx, `
		SELECT discord_user, flag, linked_at FROM links WHERE account_id = $1
	`, uuid.UUID(account)).Scan(&user, &flag, &linkedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoLink
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the account's Discord link: %w", err)
	}
	return &alertsv1.DiscordLink{
		DiscordUserId: user, CountryId: flag.String, LinkedAtUnixMs: linkedAt.UnixMilli(),
	}, nil
}
