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

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ messages.Storage = (*Store)(nil)

func (s *Store) Append(ctx context.Context, record messages.Record) error {
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO messages (id, sent_at, account_id, name, author_admin, author_id, country, ip, user_agent, text, reply_to)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
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
				SELECT id FROM messages
				WHERE sent_at >= $2 AND NOT EXISTS (SELECT 1 FROM removals WHERE removals.message_id = messages.id)
				ORDER BY seq DESC LIMIT $3
			) recent
			WHERE recent.id = $1
		)
	`, string(id), since, limit).Scan(&shown); err != nil {
		return false, fmt.Errorf("failed to read whether a message is shown: %w", err)
	}
	return shown, nil
}

func (s *Store) Message(ctx context.Context, id messages.MessageID, since time.Time, limit int) (messages.Message, error) {
	var (
		sentAt  time.Time
		account uuid.NullUUID
		name    string
		admin   bool
		country string
		text    string
		replyTo sql.NullString
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT recent.sent_at, recent.account_id, recent.name, recent.author_admin, recent.country, recent.text,
			recent.reply_to
		FROM (
			SELECT id, sent_at, account_id, name, author_admin, country, text, reply_to
			FROM messages
			WHERE sent_at >= $2 AND NOT EXISTS (SELECT 1 FROM removals WHERE removals.message_id = messages.id)
			ORDER BY seq DESC LIMIT $3
		) recent
		WHERE recent.id = $1
	`, string(id), since, limit).Scan(&sentAt, &account, &name, &admin, &country, &text, &replyTo)
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
		Author:  messages.AuthorOf(name, admin, 0, 0, messages.Title{}),
		Country: country,
		Text:    text,
		Quote:   messages.Quote{MessageID: messages.MessageID(replyTo.String)},
	}.Restore()
	if err != nil {
		return messages.Message{}, fmt.Errorf("failed to restore message %q: %w", id, err)
	}
	return message, nil
}

func (s *Store) LatestAddress(ctx context.Context, account messages.AccountID) (string, error) {
	var ip string
	err := s.db.QueryRowContext(ctx, `
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

// Removed or not: a removal is of a message that was kept.
func (s *Store) Kept(ctx context.Context, id messages.MessageID, since time.Time) (bool, error) {
	var kept bool
	if err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM messages WHERE id = $1 AND sent_at >= $2)
	`, string(id), since).Scan(&kept); err != nil {
		return false, fmt.Errorf("failed to read whether a message is kept: %w", err)
	}
	return kept, nil
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
	kept := record.TakeSnapshot()
	message := kept.Message

	return []any{
		string(message.ID),
		message.SentAt.UTC(),
		account(message.Account),
		message.Author.Name(),
		message.Author.Admin(),
		kept.AuthorID,
		message.Country,
		kept.IP,
		kept.UserAgent,
		message.Text,
		sql.NullString{String: string(message.Quote.MessageID), Valid: message.Quote.MessageID != ""},
	}
}

func account(id messages.AccountID) uuid.NullUUID {
	if id == messages.NoAccount {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: uuid.UUID(id), Valid: true}
}
