package postgres_conversation_store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ conversations.Storage = (*Store)(nil)

const foreignKeyViolation = "23503"

func (s *Store) Save(ctx context.Context, saved ...conversations.Conversation) error {
	err := s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		for _, conversation := range saved {
			if err := write(ctx, tx, conversation.TakeSnapshot()); err != nil {
				return err
			}
		}
		return nil
	})

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == foreignKeyViolation {
		return fmt.Errorf("%w: %s", notes.ErrUnknownNote, pqErr.Detail)
	}
	return err //nolint:wrapcheck // write wrapped it.
}

func write(ctx context.Context, tx cppg.Querier, kept conversations.Snapshot) error {
	note := uuid.UUID(kept.Note)
	for _, answer := range kept.Answers {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO note_answers (note_id, message_id, text, sent_at) VALUES ($1, $2, $3, $4)
			ON CONFLICT (note_id, message_id) DO UPDATE SET text = EXCLUDED.text, sent_at = EXCLUDED.sent_at
		`, note, string(answer.Mail), answer.Text, answer.At.UTC()); err != nil {
			return fmt.Errorf("failed to upsert an answer: %w", err)
		}
	}
	for _, reply := range kept.Replies {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO note_replies (note_id, message_id, sender, text, received_at) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (note_id, message_id) DO UPDATE SET
				sender = EXCLUDED.sender, text = EXCLUDED.text, received_at = EXCLUDED.received_at
		`, note, string(reply.Mail), string(reply.From), reply.Text, reply.At.UTC()); err != nil {
			return fmt.Errorf("failed to upsert a reply: %w", err)
		}
	}
	if kept.ClosedAt.IsZero() {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO note_closings (note_id, closed_at) VALUES ($1, $2)
		ON CONFLICT (note_id) DO UPDATE SET closed_at = EXCLUDED.closed_at
	`, note, kept.ClosedAt.UTC()); err != nil {
		return fmt.Errorf("failed to upsert a closing: %w", err)
	}
	return nil
}

type answer struct {
	Mail string    `json:"mail"`
	Text string    `json:"text"`
	At   time.Time `json:"at"`
}

type reply struct {
	Mail string    `json:"mail"`
	From string    `json:"from"`
	Text string    `json:"text"`
	At   time.Time `json:"at"`
}

const conversationsOf = `
	SELECT asked.id,
		COALESCE((
			SELECT json_agg(json_build_object('mail', message_id, 'text', text, 'at', sent_at) ORDER BY sent_at, message_id)
			FROM note_answers WHERE note_id = asked.id
		), '[]'),
		COALESCE((
			SELECT json_agg(json_build_object('mail', message_id, 'from', sender, 'text', text, 'at', received_at)
				ORDER BY received_at, message_id)
			FROM note_replies WHERE note_id = asked.id
		), '[]'),
		(SELECT closed_at FROM note_closings WHERE note_id = asked.id)
	FROM unnest($1::uuid[]) WITH ORDINALITY AS asked (id, position)
	ORDER BY asked.position
`

func (s *Store) Conversations(ctx context.Context, asked ...notes.NoteID) ([]conversations.Conversation, error) {
	ids := make([]string, 0, len(asked))
	for _, note := range asked {
		ids = append(ids, note.String())
	}

	rows, err := s.db.On(ctx).QueryContext(ctx, conversationsOf, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to read conversations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	found := make([]conversations.Conversation, 0, len(asked))
	for rows.Next() {
		var (
			id                uuid.UUID
			answered, replied []byte
			closed            sql.NullTime
		)
		if err := rows.Scan(&id, &answered, &replied, &closed); err != nil {
			return nil, fmt.Errorf("failed to read a conversation: %w", err)
		}
		conversation, err := restored(notes.NoteID(id), answered, replied, closed.Time)
		if err != nil {
			return nil, err
		}
		found = append(found, conversation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read conversations: %w", err)
	}
	return found, nil
}

func restored(note notes.NoteID, answered, replied []byte, closed time.Time) (conversations.Conversation, error) {
	var answers []answer
	if err := json.Unmarshal(answered, &answers); err != nil {
		return conversations.Conversation{}, fmt.Errorf("failed to decode the answers to the note %s: %w", note, err)
	}
	var replies []reply
	if err := json.Unmarshal(replied, &replies); err != nil {
		return conversations.Conversation{}, fmt.Errorf("failed to decode the replies to the note %s: %w", note, err)
	}

	snapshot := conversations.Snapshot{Note: note, ClosedAt: closed.UTC()}
	if closed.IsZero() {
		snapshot.ClosedAt = time.Time{}
	}
	for _, a := range answers {
		snapshot.Answers = append(snapshot.Answers, conversations.Answer{Text: a.Text, Mail: cpmail.MessageID(a.Mail), At: a.At.UTC()})
	}
	for _, r := range replies {
		snapshot.Replies = append(snapshot.Replies, conversations.Reply{
			Mail: cpmail.MessageID(r.Mail), From: cpmail.Address(r.From), Text: r.Text, At: r.At.UTC(),
		})
	}

	conversation, err := snapshot.Restore()
	if err != nil {
		return conversations.Conversation{}, fmt.Errorf("failed to restore the conversation of the note %s: %w", note, err)
	}
	return conversation, nil
}
