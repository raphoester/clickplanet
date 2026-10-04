package postgres_message_store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

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
		INSERT INTO messages (id, sent_at, account_id, name, author_admin, author_id, country, ip, user_agent, text)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, row(record)...); err != nil {
		return fmt.Errorf("failed to insert a message: %w", err)
	}
	return nil
}

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
		account(record.Message.Account),
		record.Message.AuthorName,
		record.Message.AuthorAdmin,
		record.AuthorID,
		record.Message.CountryID,
		record.IP,
		record.UserAgent,
		record.Message.Text,
	}
}

func account(id messages.AccountID) uuid.NullUUID {
	if id == messages.NoAccount {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: uuid.UUID(id), Valid: true}
}
