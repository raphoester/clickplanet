package messages_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
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
	DeletedName  = "[deleted]"
	DefaultLimit = 50
	MaxLimit     = 200
)

var (
	ErrNoMessage  = errors.New("no message kept under this id")
	ErrTwoCursors = errors.New("a page is before a message or after it, not both")
)

// Which page: the newest with no cursor, else the messages just before Before or just after After.
type Cursor struct {
	Before string
	After  string
}

func NewPostgresQuery(db cppg.Querier, authors Authors, clock cptime.Clock, retention time.Duration) *PostgresQuery {
	return &PostgresQuery{db: db, authors: authors, clock: clock, retention: retention}
}

type PostgresQuery struct {
	db        cppg.Querier
	authors   Authors
	clock     cptime.Clock
	retention time.Duration
}

const keptSeq = `
	SELECT seq
	FROM messages
	WHERE id = $1 AND sent_at >= $2
	ORDER BY seq DESC
	LIMIT 1
`

// $1 is the start of the retention, $2 the limit and $3 the cursor's seq.
const (
	older = `
		SELECT seq, id, account_id, name, text, sent_at, country, reply_to
		FROM messages
		WHERE sent_at >= $1 AND seq < $3
		ORDER BY seq DESC
		LIMIT $2
	`
	newer = `
		SELECT seq, id, account_id, name, text, sent_at, country, reply_to
		FROM messages
		WHERE sent_at >= $1 AND seq > $3
		ORDER BY seq
		LIMIT $2
	`
	projection = `)
	SELECT
		page.id, page.account_id, page.name, page.text, page.sent_at, page.country, removals.removed_at,
		COALESCE(quoted.id, ''), quoted.account_id, COALESCE(quoted.name, ''), COALESCE(quoted.text, ''),
		quoted.removed_at
	FROM page
	LEFT JOIN removals ON removals.message_id = page.id
	LEFT JOIN LATERAL (
		SELECT messages.id, messages.account_id, messages.name, messages.text, removals.removed_at
		FROM messages
		LEFT JOIN removals ON removals.message_id = messages.id
		WHERE messages.id = page.reply_to AND messages.sent_at >= $1
		ORDER BY messages.seq DESC
		LIMIT 1
	) quoted ON true
	ORDER BY page.seq
`
	olderPage = `WITH page AS (` + older + projection
	newerPage = `WITH page AS (` + newer + projection
)

type row struct {
	message *chatv1.KeptMessage
	author  author
	quoted  author
}

// Who sent a message: an account, or the name it was sent under before accounts.
type author struct {
	account cpsession.AccountID
	name    string
}

// A zero limit is DefaultLimit, and none passes MaxLimit.
func (q *PostgresQuery) Messages(ctx context.Context, cursor Cursor, limit uint32) (*chatv1.ListMessagesResponse, error) {
	if cursor.Before != "" && cursor.After != "" {
		return nil, ErrTwoCursors
	}

	since := q.clock.Now().Add(-q.retention)
	statement, from, err := q.window(ctx, cursor, since)
	if err != nil {
		return nil, err
	}

	rows, err := q.rows(ctx, statement, since, limitOf(limit), from)
	if err != nil {
		return nil, err
	}

	named, err := q.authors.Authors(ctx, everyone(rows))
	if err != nil {
		return nil, fmt.Errorf("failed to read who wrote a page of messages: %w", err)
	}

	answer := &chatv1.ListMessagesResponse{Messages: make([]*chatv1.KeptMessage, 0, len(rows))}
	for _, row := range rows {
		row.message.AuthorName = nameOf(row.author, named)
		if row.message.GetReplyTo() != nil {
			row.message.ReplyTo.AuthorName = nameOf(row.quoted, named)
		}
		answer.Messages = append(answer.Messages, row.message)
	}
	return answer, nil
}

// The statement that reads the page, and the seq it reads from.
func (q *PostgresQuery) window(ctx context.Context, cursor Cursor, since time.Time) (string, int64, error) {
	switch {
	case cursor.Before != "":
		seq, err := q.seqOf(ctx, cursor.Before, since)
		return olderPage, seq, err
	case cursor.After != "":
		seq, err := q.seqOf(ctx, cursor.After, since)
		return newerPage, seq, err
	default:
		return olderPage, math.MaxInt64, nil
	}
}

func (q *PostgresQuery) seqOf(ctx context.Context, id string, since time.Time) (int64, error) {
	var seq int64
	err := q.db.QueryRowContext(ctx, keptSeq, id, since).Scan(&seq)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("%w: %q", ErrNoMessage, id)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to read message %q: %w", id, err)
	}
	return seq, nil
}

func (q *PostgresQuery) rows(ctx context.Context, statement string, since time.Time, limit int, from int64) ([]row, error) {
	found, err := q.db.QueryContext(ctx, statement, since, limit, from)
	if err != nil {
		return nil, fmt.Errorf("failed to read a page of messages: %w", err)
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
			&each.message.MessageId, &account, &each.author.name, &each.message.Text, &sentAt,
			&each.message.CountryId, &removedAt,
			&quote.MessageId, &quotedAccount, &each.quoted.name, &quote.Text, &quotedRemovedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to read a message of a page: %w", err)
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
		return nil, fmt.Errorf("failed to read a page of messages: %w", err)
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

func limitOf(asked uint32) int {
	switch {
	case asked == 0:
		return DefaultLimit
	case asked > MaxLimit:
		return MaxLimit
	default:
		return int(asked)
	}
}
