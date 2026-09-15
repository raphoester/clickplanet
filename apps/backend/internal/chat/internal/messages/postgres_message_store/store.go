// Package postgres_message_store keeps every chat message in the chat schema: one row per message, sender included.
package postgres_message_store

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"time"

	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

func (s *Store) Insert(ctx context.Context, record messages.Record) error {
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO messages (id, sent_at, name, tag, author_id, country, ip, user_agent, text)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, row(record)...); err != nil {
		return fmt.Errorf("failed to insert a message: %w", err)
	}
	return nil
}

// InsertAll writes records in one transaction, in order: all of them or none.
func (s *Store) InsertAll(ctx context.Context, records []messages.Record) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to insert messages: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx,
		pq.CopyIn("messages", "id", "sent_at", "name", "tag", "author_id", "country", "ip", "user_agent", "text"))
	if err != nil {
		return fmt.Errorf("failed to start copying messages: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, record := range records {
		if _, err := stmt.ExecContext(ctx, row(record)...); err != nil {
			return fmt.Errorf("failed to copy a message: %w", err)
		}
	}
	if _, err := stmt.ExecContext(ctx); err != nil {
		return fmt.Errorf("failed to copy messages: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit messages: %w", err)
	}
	return nil
}

func (s *Store) IsEmpty(ctx context.Context) (bool, error) {
	var empty bool
	if err := s.db.QueryRowContext(ctx, `SELECT NOT EXISTS (SELECT 1 FROM messages)`).Scan(&empty); err != nil {
		return false, fmt.Errorf("failed to count messages: %w", err)
	}
	return empty, nil
}

// Recent is the newest limit messages sent at or after since, oldest first.
func (s *Store) Recent(ctx context.Context, since time.Time, limit int) ([]messages.Message, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, sent_at, name, tag, country, text
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
		var message messages.Message
		if err := rows.Scan(
			&message.ID, &message.SentAt, &message.AuthorName, &message.AuthorTag, &message.CountryID, &message.Text,
		); err != nil {
			return nil, fmt.Errorf("failed to scan a message: %w", err)
		}
		message.SentAt = message.SentAt.UTC()
		recent = append(recent, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read messages: %w", err)
	}

	slices.Reverse(recent)
	return recent, nil
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
		record.Message.ID,
		record.Message.SentAt.UTC(),
		record.Message.AuthorName,
		record.Message.AuthorTag,
		record.AuthorID,
		record.Message.CountryID,
		record.IP,
		record.UserAgent,
		record.Message.Text,
	}
}
