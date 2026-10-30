package mute_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewPostgresQuery(db cppg.Conn, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock}
}

type PostgresQuery struct {
	db    cppg.Conn
	clock cptime.Clock
}

const latestAddress = `SELECT ip FROM messages WHERE account_id = $1 ORDER BY seq DESC LIMIT 1`

const mute = `
	SELECT account_id, muted_until
	FROM mutes
	WHERE muted_until > $3 AND (account_id = $1 OR scope = $2)
	ORDER BY muted_until DESC, id
	LIMIT 1
`

func (q *PostgresQuery) Mute(ctx context.Context, account messages.AccountID) (*chatv1.GetMuteResponse, error) {
	var ip string
	err := q.db.On(ctx).QueryRowContext(ctx, latestAddress, uuid.UUID(account)).Scan(&ip)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to read where the account posts from: %w", err)
	}
	var scope sql.NullString
	scope.String, scope.Valid = cpipscope.Parse(ip)

	var (
		given uuid.UUID
		until time.Time
	)
	err = q.db.On(ctx).QueryRowContext(ctx, mute, uuid.UUID(account), scope, q.clock.Now().UTC()).Scan(&given, &until)
	if errors.Is(err, sql.ErrNoRows) {
		return &chatv1.GetMuteResponse{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the mute: %w", err)
	}
	return &chatv1.GetMuteResponse{MutedUntil: timestamppb.New(until), MutedAccountId: given.String()}, nil
}
