package postgres_message_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ messages.Storage = (*Store)(nil)

func (s *Store) Append(ctx context.Context, record messages.Record) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		WITH kept AS (
			INSERT INTO messages (id, sent_at, account_id, country, ip, user_agent, text, reply_to, room)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING seq
		)
		INSERT INTO gifs (message_seq, url, width, height)
		SELECT seq, $10, $11, $12 FROM kept WHERE $10 <> ''
	`, row(record)...); err != nil {
		return fmt.Errorf("failed to insert a message: %w", err)
	}
	return nil
}

func (s *Store) Message(ctx context.Context, id messages.MessageID, since time.Time, limit int) (messages.Message, error) {
	var (
		sentAt  time.Time
		account uuid.NullUUID
		room    string
		country string
		text    string
		replyTo sql.NullString
		gif     messages.Gif
	)
	err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT recent.sent_at, recent.account_id, recent.room, recent.country, recent.text, recent.reply_to,
			COALESCE(gifs.url, ''), COALESCE(gifs.width, 0), COALESCE(gifs.height, 0)
		FROM (
			SELECT seq, id, sent_at, account_id, room, country, text, reply_to
			FROM messages
			WHERE room = (SELECT room FROM messages WHERE id = $1 ORDER BY seq DESC LIMIT 1) AND sent_at >= $2
			ORDER BY seq DESC LIMIT $3
		) recent
		LEFT JOIN gifs ON gifs.message_seq = recent.seq
		WHERE recent.id = $1 AND NOT EXISTS (SELECT 1 FROM removals WHERE removals.message_id = recent.id)
	`, string(id), since, limit).Scan(&sentAt, &account, &room, &country, &text, &replyTo,
		&gif.URL, &gif.Width, &gif.Height)
	if errors.Is(err, sql.ErrNoRows) {
		return messages.Message{}, messages.ErrNotShown
	}
	if err != nil {
		return messages.Message{}, fmt.Errorf("failed to read a shown message: %w", err)
	}

	message, err := messages.Snapshot{
		ID:      id,
		SentAt:  sentAt.UTC(),
		Account: messages.AccountID(account.UUID),
		Room:    messages.Room(room),
		Country: country,
		Text:    text,
		Gif:     gif,
		Quote:   messages.Quote{MessageID: messages.MessageID(replyTo.String)},
	}.Restore()
	if err != nil {
		return messages.Message{}, fmt.Errorf("failed to restore message %q: %w", id, err)
	}
	return message, nil
}

func (s *Store) LatestAddress(ctx context.Context, account messages.AccountID) (string, error) {
	var ip string
	err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT ip FROM messages WHERE account_id = $1 ORDER BY seq DESC LIMIT 1
	`, uuid.UUID(account)).Scan(&ip)
	if errors.Is(err, sql.ErrNoRows) {
		return "", messages.ErrNoMessage
	}
	if err != nil {
		return "", fmt.Errorf("failed to read the latest address of an account: %w", err)
	}
	return ip, nil
}

// Removed or not: a removal is of a message that was kept. ErrNotKept past the retention.
func (s *Store) KeptMessage(ctx context.Context, id messages.MessageID, since time.Time) (messages.Message, error) {
	var (
		sentAt  time.Time
		account uuid.NullUUID
		room    string
		country string
		text    string
		gif     messages.Gif
	)
	err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT messages.sent_at, messages.account_id, messages.room, messages.country, messages.text,
			COALESCE(gifs.url, ''), COALESCE(gifs.width, 0), COALESCE(gifs.height, 0)
		FROM messages
		LEFT JOIN gifs ON gifs.message_seq = messages.seq
		WHERE messages.id = $1 AND messages.sent_at >= $2
		ORDER BY messages.seq DESC
		LIMIT 1
	`, string(id), since).Scan(&sentAt, &account, &room, &country, &text, &gif.URL, &gif.Width, &gif.Height)
	if errors.Is(err, sql.ErrNoRows) {
		return messages.Message{}, fmt.Errorf("%w: %q", messages.ErrNotKept, id)
	}
	if err != nil {
		return messages.Message{}, fmt.Errorf("failed to read a kept message: %w", err)
	}

	message, err := messages.Snapshot{
		ID:      id,
		SentAt:  sentAt.UTC(),
		Account: messages.AccountID(account.UUID),
		Room:    messages.Room(room),
		Country: country,
		Text:    text,
		Gif:     gif,
	}.Restore()
	if err != nil {
		return messages.Message{}, fmt.Errorf("failed to restore message %q: %w", id, err)
	}
	return message, nil
}

func (s *Store) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM messages WHERE sent_at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old messages: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count deleted messages: %w", err)
	}
	return deleted, nil
}

func row(record messages.Record) []any {
	kept := record.TakeSnapshot()
	message := kept.Message

	return []any{
		string(message.ID),
		message.SentAt.UTC(),
		account(message.Account),
		message.Country,
		kept.IP,
		kept.UserAgent,
		message.Text,
		sql.NullString{String: string(message.Quote.MessageID), Valid: message.Quote.MessageID != ""},
		string(message.Room),
		message.Gif.URL,
		message.Gif.Width,
		message.Gif.Height,
	}
}

func account(id messages.AccountID) uuid.NullUUID {
	if id == messages.NoAccount {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: uuid.UUID(id), Valid: true}
}
