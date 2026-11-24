package letters_query

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/lettermessage"
)

type Filter struct {
	// Nil for every account.
	Account *notes.AccountID
	// Nil for every submission.
	Submission *letters.SubmissionID
	// The letters the player replied to since the last mail and the last close.
	OpenOnly bool
	Limit    int
}

func NewPostgresQuery(db cppg.Conn) PostgresQuery {
	return PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

const query = `
	WITH found AS (
		SELECT l.id, l.submission_id, l.account_id, l.address, l.subject,
			COALESCE((
				SELECT json_agg(json_build_object('mail', message_id, 'text', text, 'at', sent_at) ORDER BY sent_at, message_id)
				FROM letter_mails WHERE letter_id = l.id
			), '[]') AS mails,
			COALESCE((
				SELECT json_agg(json_build_object('mail', message_id, 'from', sender, 'text', text, 'at', received_at)
					ORDER BY received_at, message_id)
				FROM letter_replies WHERE letter_id = l.id
			), '[]') AS replies,
			(SELECT closed_at FROM letter_closings WHERE letter_id = l.id) AS closed_at,
			(SELECT max(sent_at) FROM letter_mails WHERE letter_id = l.id) AS last_sent,
			(SELECT max(received_at) FROM letter_replies WHERE letter_id = l.id) AS last_reply
		FROM letters l
		WHERE ($1::uuid IS NULL OR l.account_id = $1) AND ($4::uuid IS NULL OR l.submission_id = $4)
	)
	SELECT id, submission_id, account_id, address, subject, mails, replies, closed_at FROM found
	WHERE NOT $2 OR (last_reply > last_sent AND (closed_at IS NULL OR last_reply > closed_at))
	ORDER BY GREATEST(last_sent, COALESCE(last_reply, last_sent)) DESC, id
	LIMIT $3`

type mail struct {
	Mail string    `json:"mail"`
	From string    `json:"from"`
	Text string    `json:"text"`
	At   time.Time `json:"at"`
}

func (q PostgresQuery) Letters(ctx context.Context, filter Filter) (*supportv1.ListLettersResponse, error) {
	var account, submission uuid.NullUUID
	if filter.Account != nil {
		account = uuid.NullUUID{UUID: uuid.UUID(*filter.Account), Valid: true}
	}
	if filter.Submission != nil {
		submission = uuid.NullUUID{UUID: uuid.UUID(*filter.Submission), Valid: true}
	}
	rows, err := q.db.On(ctx).QueryContext(ctx, query, account, filter.OpenOnly, filter.Limit, submission)
	if err != nil {
		return nil, fmt.Errorf("failed to read the letters: %w", err)
	}
	defer func() { _ = rows.Close() }()

	found := &supportv1.ListLettersResponse{}
	for rows.Next() {
		var (
			id, about     uuid.UUID
			owner         uuid.NullUUID
			address       string
			row           letters.Snapshot
			written, said []byte
			closed        sql.NullTime
		)
		if err := rows.Scan(&id, &about, &owner, &address, &row.Subject, &written, &said, &closed); err != nil {
			return nil, fmt.Errorf("failed to read a letter: %w", err)
		}
		row.ID, row.Submission, row.To = letters.LetterID(id), letters.SubmissionID(about), cpmail.Address(address)
		if owner.Valid {
			row.Account = notes.AccountID(owner.UUID)
		}
		if closed.Valid {
			row.ClosedAt = closed.Time.UTC()
		}
		var sent, replies []mail
		if err := json.Unmarshal(written, &sent); err != nil {
			return nil, fmt.Errorf("failed to decode the mails of the letter %s: %w", row.ID, err)
		}
		if err := json.Unmarshal(said, &replies); err != nil {
			return nil, fmt.Errorf("failed to decode the replies to the letter %s: %w", row.ID, err)
		}
		for _, m := range sent {
			row.Sent = append(row.Sent, letters.Sent{Text: m.Text, Mail: cpmail.MessageID(m.Mail), At: m.At.UTC()})
		}
		for _, m := range replies {
			row.Replies = append(row.Replies, conversations.Reply{
				Mail: cpmail.MessageID(m.Mail), From: cpmail.Address(m.From), Text: m.Text, At: m.At.UTC(),
			})
		}
		found.Letters = append(found.Letters, lettermessage.Letter(row))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the letters: %w", err)
	}
	return found, nil
}
