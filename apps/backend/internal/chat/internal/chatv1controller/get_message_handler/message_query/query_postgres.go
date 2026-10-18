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
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Authors interface {
	Authors(ctx context.Context, accounts []cpsession.AccountID) (map[cpsession.AccountID]*playerv1.Author, error)
}

const (
	DeletedName = "[deleted]"
	MostAround  = 50
)

var ErrNoMessage = errors.New("no message kept under this id")

func NewPostgresQuery(db cppg.Querier, authors Authors, clock cptime.Clock, retention time.Duration) *PostgresQuery {
	return &PostgresQuery{db: db, authors: authors, clock: clock, retention: retention}
}

type PostgresQuery struct {
	db        cppg.Querier
	authors   Authors
	clock     cptime.Clock
	retention time.Duration
}

// side is -1 for a message sent before the one asked for, 0 for that one and 1 for one sent after it.
const around = `
	WITH asked AS (
		SELECT seq
		FROM messages
		WHERE id = $1 AND sent_at >= $2
		ORDER BY seq DESC
		LIMIT 1
	),
	around AS (
		SELECT 0 AS side, seq, id, account_id, name, text, sent_at, country, reply_to
		FROM messages
		WHERE seq = (SELECT seq FROM asked)
		UNION ALL (
			SELECT -1, seq, id, account_id, name, text, sent_at, country, reply_to
			FROM messages
			WHERE seq < (SELECT seq FROM asked) AND sent_at >= $2
			ORDER BY seq DESC
			LIMIT $3
		)
		UNION ALL (
			SELECT 1, seq, id, account_id, name, text, sent_at, country, reply_to
			FROM messages
			WHERE seq > (SELECT seq FROM asked)
			ORDER BY seq
			LIMIT $4
		)
	)
	SELECT
		around.side, around.id, around.account_id, around.name, around.text, around.sent_at, around.country,
		removals.removed_at,
		COALESCE(quoted.id, ''), quoted.account_id, COALESCE(quoted.name, ''), COALESCE(quoted.text, ''),
		quoted.removed_at
	FROM around
	LEFT JOIN removals ON removals.message_id = around.id
	LEFT JOIN LATERAL (
		SELECT messages.id, messages.account_id, messages.name, messages.text, removals.removed_at
		FROM messages
		LEFT JOIN removals ON removals.message_id = messages.id
		WHERE messages.id = around.reply_to AND messages.sent_at >= $2
		ORDER BY messages.seq DESC
		LIMIT 1
	) quoted ON true
	ORDER BY around.seq
`

type row struct {
	side    int
	message *chatv1.KeptMessage
	author  author
	quoted  author
}

// Who sent a message: an account, or the name it was sent under before accounts.
type author struct {
	account cpsession.AccountID
	name    string
}

// before and after are capped at MostAround.
func (q *PostgresQuery) Message(ctx context.Context, id string, before, after uint32) (*chatv1.GetMessageResponse, error) {
	rows, err := q.rows(ctx, id, before, after)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%w: %q", ErrNoMessage, id)
	}

	named, err := q.authors.Authors(ctx, everyone(rows))
	if err != nil {
		return nil, fmt.Errorf("failed to read who wrote around message %q: %w", id, err)
	}

	answer := &chatv1.GetMessageResponse{}
	for _, row := range rows {
		row.message.AuthorName = nameOf(row.author, named)
		if row.message.GetReplyTo() != nil {
			row.message.ReplyTo.AuthorName = nameOf(row.quoted, named)
		}

		switch {
		case row.side < 0:
			answer.Before = append(answer.Before, row.message)
		case row.side > 0:
			answer.After = append(answer.After, row.message)
		default:
			answer.MessageId = row.message.GetMessageId()
			answer.AccountId = row.message.GetAccountId()
			answer.AuthorName = row.message.GetAuthorName()
			answer.Text = row.message.GetText()
			answer.SentAt = row.message.GetSentAt()
			answer.CountryId = row.message.GetCountryId()
			answer.RemovedAt = row.message.GetRemovedAt()
			answer.ReplyTo = row.message.GetReplyTo()
		}
	}
	return answer, nil
}

func (q *PostgresQuery) rows(ctx context.Context, id string, before, after uint32) ([]row, error) {
	found, err := q.db.QueryContext(ctx, around,
		id, q.clock.Now().Add(-q.retention), min(before, MostAround), min(after, MostAround))
	if err != nil {
		return nil, fmt.Errorf("failed to read message %q and those around it: %w", id, err)
	}
	defer func() { _ = found.Close() }()

	var rows []row
	for found.Next() {
		var (
			each            = row{message: &chatv1.KeptMessage{}}
			account         uuid.NullUUID
			sentAt          time.Time
			removedAt       sql.NullTime
			quote           = &chatv1.KeptQuote{}
			quotedAccount   uuid.NullUUID
			quotedRemovedAt sql.NullTime
		)
		if err := found.Scan(
			&each.side, &each.message.MessageId, &account, &each.author.name, &each.message.Text, &sentAt,
			&each.message.CountryId, &removedAt,
			&quote.MessageId, &quotedAccount, &each.quoted.name, &quote.Text, &quotedRemovedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to read a message around %q: %w", id, err)
		}

		each.message.SentAt = timestamppb.New(sentAt)
		if account.Valid {
			each.author.account = cpsession.AccountID(account.UUID)
			each.message.AccountId = account.UUID.String()
		}
		if removedAt.Valid {
			each.message.RemovedAt = timestamppb.New(removedAt.Time)
		}
		if quote.GetMessageId() != "" {
			if quotedAccount.Valid {
				each.quoted.account = cpsession.AccountID(quotedAccount.UUID)
			}
			if quotedRemovedAt.Valid {
				quote.RemovedAt = timestamppb.New(quotedRemovedAt.Time)
			}
			each.message.ReplyTo = quote
		}
		rows = append(rows, each)
	}
	if err := found.Err(); err != nil {
		return nil, fmt.Errorf("failed to read message %q and those around it: %w", id, err)
	}
	return rows, nil
}

func everyone(rows []row) []cpsession.AccountID {
	seen := cpcolls.NewSet[cpsession.AccountID]()
	accounts := make([]cpsession.AccountID, 0, len(rows))
	for _, row := range rows {
		for _, account := range []cpsession.AccountID{row.author.account, row.quoted.account} {
			if account == cpsession.NoAccount || seen.Contains(account) {
				continue
			}
			seen.Add(account)
			accounts = append(accounts, account)
		}
	}
	return accounts
}

func nameOf(author author, named map[cpsession.AccountID]*playerv1.Author) string {
	if author.account == cpsession.NoAccount {
		return author.name
	}
	if known, ok := named[author.account]; ok {
		return known.GetName()
	}
	return DeletedName
}
