package message_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrNoMessage = errors.New("no message kept under this id")

func NewPostgresQuery(db cppg.Conn, clock cptime.Clock, retention time.Duration) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock, retention: retention}
}

type PostgresQuery struct {
	db        cppg.Conn
	clock     cptime.Clock
	retention time.Duration
}

const message = `
	SELECT messages.account_id, messages.text, messages.sent_at, messages.country, removals.removed_at
	FROM messages
	LEFT JOIN removals ON removals.message_id = messages.id
	WHERE messages.id = $1 AND messages.sent_at >= $2
	ORDER BY messages.seq DESC
	LIMIT 1
`

func (q *PostgresQuery) Message(ctx context.Context, id string) (*chatv1.GetMessageResponse, error) {
	var (
		account   uuid.NullUUID
		text      string
		sentAt    time.Time
		country   string
		removedAt sql.NullTime
	)
	err := q.db.On(ctx).QueryRowContext(ctx, message, id, q.clock.Now().Add(-q.retention)).
		Scan(&account, &text, &sentAt, &country, &removedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %q", ErrNoMessage, id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the message: %w", err)
	}

	answer := &chatv1.GetMessageResponse{
		MessageId: id,
		Text:      text,
		SentAt:    timestamppb.New(sentAt),
		CountryId: country,
	}
	if account.Valid {
		answer.AccountId = account.UUID.String()
	}
	if removedAt.Valid {
		answer.RemovedAt = timestamppb.New(removedAt.Time)
	}
	return answer, nil
}
