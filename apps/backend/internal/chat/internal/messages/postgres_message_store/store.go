// Package postgres_message_store keeps every chat message in chat.messages, sender included. It is the chat's only
// copy: every read and write goes to postgres.
package postgres_message_store

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ messages.Storage = (*Store)(nil)

func (s *Store) Append(ctx context.Context, record messages.Record) error {
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO messages (id, sent_at, name, author_admin, author_id, country, ip, user_agent, text)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, row(record)...); err != nil {
		return fmt.Errorf("failed to insert a message: %w", err)
	}
	return nil
}

// Recent is the newest limit messages sent at or after since, oldest first.
func (s *Store) Recent(ctx context.Context, since time.Time, limit int) ([]messages.Message, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, sent_at, name, author_admin, country, text
		FROM messages
		WHERE sent_at >= $1
		ORDER BY seq DESC
		LIMIT $2
	`, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to read messages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var recent []messages.Message
	for rows.Next() {
		var (
			message messages.Message
			id      string
		)
		if err := rows.Scan(
			&id, &message.SentAt, &message.AuthorName, &message.AuthorAdmin,
			&message.CountryID, &message.Text,
		); err != nil {
			return nil, fmt.Errorf("failed to scan a message: %w", err)
		}
		message.ID = messages.MessageID(id)
		message.SentAt = message.SentAt.UTC()
		recent = append(recent, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read messages: %w", err)
	}

	slices.Reverse(recent)
	return recent, nil
}

// Shown asks the same question as Recent about one message. The newest limit rows are a walk down the primary key.
func (s *Store) Shown(ctx context.Context, id messages.MessageID, since time.Time, limit int) (bool, error) {
	var shown bool
	if err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM (
				SELECT id FROM messages WHERE sent_at >= $2 ORDER BY seq DESC LIMIT $3
			) recent
			WHERE recent.id = $1
		)
	`, string(id), since, limit).Scan(&shown); err != nil {
		return false, fmt.Errorf("failed to read whether a message is shown: %w", err)
	}
	return shown, nil
}

// DeleteBefore removes every message sent before cutoff and says how many it removed.
func (s *Store) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE sent_at < $1`, cutoff)
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
	return []any{
		string(record.Message.ID),
		record.Message.SentAt.UTC(),
		record.Message.AuthorName,
		record.Message.AuthorAdmin,
		record.AuthorID,
		record.Message.CountryID,
		record.IP,
		record.UserAgent,
		record.Message.Text,
	}
}
