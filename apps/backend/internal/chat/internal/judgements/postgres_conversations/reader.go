package postgres_conversations

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// How much the judge reads, and so how much each judgement costs: a few thousand tokens.
const (
	Before = 15
	After  = 10
	Own    = 20
)

func New(db cppg.Conn) *Reader {
	return &Reader{db: db}
}

// The chat as the judge reads it, removed messages included, from chat.messages, chat.gifs and chat.removals.
type Reader struct {
	db cppg.Conn
}

const projection = `
	SELECT messages.seq, messages.id, messages.account_id, messages.room, messages.sent_at, messages.text,
		COALESCE(gifs.url, ''), COALESCE(gifs.width, 0), COALESCE(gifs.height, 0),
		EXISTS (SELECT 1 FROM removals WHERE removals.message_id = messages.id)
	FROM messages
	LEFT JOIN gifs ON gifs.message_seq = messages.seq
`

// messages.ErrNotKept for a message named that is not kept since then.
func (r *Reader) Conversation(
	ctx context.Context, message messages.MessageID, account messages.AccountID, since time.Time,
) (judgements.Conversation, error) {
	var conversation judgements.Conversation
	if message != judgements.NoMessage {
		reported, seq, err := r.reported(ctx, message, since)
		if err != nil {
			return judgements.Conversation{}, err
		}
		conversation.Reported = &reported
		if conversation.Before, err = r.lines(ctx, true, projection+`
			WHERE messages.room = $1 AND messages.seq < $2 AND messages.sent_at >= $3
			ORDER BY messages.seq DESC LIMIT $4
		`, string(reported.Room), seq, since, Before); err != nil {
			return judgements.Conversation{}, fmt.Errorf("failed to read what came before the message: %w", err)
		}
		if conversation.After, err = r.lines(ctx, false, projection+`
			WHERE messages.room = $1 AND messages.seq > $2
			ORDER BY messages.seq LIMIT $3
		`, string(reported.Room), seq, After); err != nil {
			return judgements.Conversation{}, fmt.Errorf("failed to read what came after the message: %w", err)
		}
	}

	own, err := r.lines(ctx, true, projection+`
		WHERE messages.account_id = $1 AND messages.sent_at >= $2
		ORDER BY messages.seq DESC LIMIT $3
	`, uuid.UUID(account), since, Own)
	if err != nil {
		return judgements.Conversation{}, fmt.Errorf("failed to read the player's own messages: %w", err)
	}
	conversation.Own = own
	return conversation, nil
}

func (r *Reader) reported(ctx context.Context, message messages.MessageID, since time.Time) (judgements.Line, int64, error) {
	lines, seqs, err := r.scan(ctx, projection+`
		WHERE messages.id = $1 AND messages.sent_at >= $2 ORDER BY messages.seq DESC LIMIT 1
	`, string(message), since)
	if err != nil {
		return judgements.Line{}, 0, fmt.Errorf("failed to read the message reported: %w", err)
	}
	if len(lines) == 0 {
		return judgements.Line{}, 0, fmt.Errorf("%w: %q", messages.ErrNotKept, message)
	}
	return lines[0], seqs[0], nil
}

// Oldest first, whichever way the statement walks.
func (r *Reader) lines(ctx context.Context, newestFirst bool, query string, args ...any) ([]judgements.Line, error) {
	lines, _, err := r.scan(ctx, query, args...)
	if newestFirst {
		slices.Reverse(lines)
	}
	return lines, err
}

func (r *Reader) scan(ctx context.Context, query string, args ...any) ([]judgements.Line, []int64, error) {
	rows, err := r.db.On(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read the chat: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var (
		lines []judgements.Line
		seqs  []int64
	)
	for rows.Next() {
		var (
			seq     int64
			id      string
			account uuid.NullUUID
			room    string
			line    judgements.Line
		)
		if err := rows.Scan(&seq, &id, &account, &room, &line.SentAt, &line.Text,
			&line.Gif.URL, &line.Gif.Width, &line.Gif.Height, &line.Removed); err != nil {
			return nil, nil, fmt.Errorf("failed to read a message: %w", err)
		}
		line.ID, line.Account, line.Room, line.SentAt = messages.MessageID(id), messages.AccountID(account.UUID),
			messages.Room(room), line.SentAt.UTC()
		lines, seqs = append(lines, line), append(seqs, seq)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to read the chat: %w", err)
	}
	return lines, seqs, nil
}
