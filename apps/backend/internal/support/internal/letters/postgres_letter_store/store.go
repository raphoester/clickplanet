package postgres_letter_store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ letters.Storage = (*Store)(nil)

const columns = `l.id, l.account_id, l.address, l.subject,
	COALESCE((
		SELECT json_agg(json_build_object('mail', message_id, 'text', text, 'at', sent_at) ORDER BY sent_at, message_id)
		FROM letter_mails WHERE letter_id = l.id
	), '[]'),
	COALESCE((
		SELECT json_agg(json_build_object('mail', message_id, 'from', sender, 'text', text, 'at', received_at)
			ORDER BY received_at, message_id)
		FROM letter_replies WHERE letter_id = l.id
	), '[]'),
	(SELECT closed_at FROM letter_closings WHERE letter_id = l.id)`

func (s *Store) Letter(ctx context.Context, id letters.LetterID) (letters.Letter, error) {
	row := s.db.On(ctx).QueryRowContext(ctx, `SELECT `+columns+` FROM letters l WHERE l.id = $1`, uuid.UUID(id))
	letter, err := letterOf(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return letters.Letter{}, fmt.Errorf("%s: %w", id, letters.ErrUnknownLetter)
	}
	return letter, err
}

func (s *Store) To(ctx context.Context, account notes.AccountID) ([]letters.Letter, error) {
	return s.read(ctx, `SELECT `+columns+` FROM letters l WHERE l.account_id = $1 ORDER BY l.written_at, l.id`,
		uuid.UUID(account))
}

func (s *Store) WrittenBefore(ctx context.Context, at time.Time) ([]letters.Letter, error) {
	return s.read(ctx, `SELECT `+columns+` FROM letters l WHERE l.account_id IS NOT NULL AND l.written_at < $1
		ORDER BY l.written_at, l.id LIMIT 500`, at.UTC())
}

func (s *Store) read(ctx context.Context, query string, args ...any) ([]letters.Letter, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read letters: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var found []letters.Letter
	for rows.Next() {
		letter, err := letterOf(rows.Scan)
		if err != nil {
			return nil, err
		}
		found = append(found, letter)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read letters: %w", err)
	}
	return found, nil
}

type mail struct {
	Mail string    `json:"mail"`
	From string    `json:"from"`
	Text string    `json:"text"`
	At   time.Time `json:"at"`
}

func letterOf(scan func(dest ...any) error) (letters.Letter, error) {
	var (
		id            uuid.UUID
		account       uuid.NullUUID
		address       string
		kept          letters.Snapshot
		written, said []byte
		closed        sql.NullTime
	)
	if err := scan(&id, &account, &address, &kept.Subject, &written, &said, &closed); err != nil {
		return letters.Letter{}, fmt.Errorf("failed to read a letter: %w", err)
	}
	kept.ID, kept.To = letters.LetterID(id), cpmail.Address(address)
	if account.Valid {
		kept.Account = notes.AccountID(account.UUID)
	}
	if closed.Valid {
		kept.ClosedAt = closed.Time.UTC()
	}

	var sent, replies []mail
	if err := json.Unmarshal(written, &sent); err != nil {
		return letters.Letter{}, fmt.Errorf("failed to decode the mails of the letter %s: %w", kept.ID, err)
	}
	if err := json.Unmarshal(said, &replies); err != nil {
		return letters.Letter{}, fmt.Errorf("failed to decode the replies to the letter %s: %w", kept.ID, err)
	}
	for _, m := range sent {
		kept.Sent = append(kept.Sent, letters.Sent{Text: m.Text, Mail: cpmail.MessageID(m.Mail), At: m.At.UTC()})
	}
	for _, m := range replies {
		kept.Replies = append(kept.Replies, conversations.Reply{
			Mail: cpmail.MessageID(m.Mail), From: cpmail.Address(m.From), Text: m.Text, At: m.At.UTC(),
		})
	}

	letter, err := kept.Restore()
	if err != nil {
		return letters.Letter{}, fmt.Errorf("failed to restore the letter %s: %w", kept.ID, err)
	}
	return letter, nil
}

func (s *Store) Save(ctx context.Context, saved ...letters.Letter) error {
	return s.db.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // write wraps each.
		tx := s.db.On(ctx)
		for _, letter := range saved {
			if err := write(ctx, tx, letter.TakeSnapshot()); err != nil {
				return err
			}
		}
		return nil
	})
}

func write(ctx context.Context, tx cppg.Querier, kept letters.Snapshot) error {
	id := uuid.UUID(kept.ID)
	var account uuid.NullUUID
	if kept.Account != (notes.AccountID{}) {
		account = uuid.NullUUID{UUID: uuid.UUID(kept.Account), Valid: true}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO letters (id, account_id, address, subject, written_at) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET account_id = EXCLUDED.account_id, address = EXCLUDED.address
	`, id, account, string(kept.To), kept.Subject, kept.Sent[0].At.UTC()); err != nil {
		return fmt.Errorf("failed to upsert the letter %s: %w", kept.ID, err)
	}
	for _, sent := range kept.Sent {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO letter_mails (letter_id, message_id, text, sent_at) VALUES ($1, $2, $3, $4)
			ON CONFLICT (letter_id, message_id) DO UPDATE SET text = EXCLUDED.text, sent_at = EXCLUDED.sent_at
		`, id, string(sent.Mail), sent.Text, sent.At.UTC()); err != nil {
			return fmt.Errorf("failed to upsert a mail of the letter %s: %w", kept.ID, err)
		}
	}
	for _, reply := range kept.Replies {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO letter_replies (letter_id, message_id, sender, text, received_at) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (letter_id, message_id) DO UPDATE SET
				sender = EXCLUDED.sender, text = EXCLUDED.text, received_at = EXCLUDED.received_at
		`, id, string(reply.Mail), string(reply.From), reply.Text, reply.At.UTC()); err != nil {
			return fmt.Errorf("failed to upsert a reply to the letter %s: %w", kept.ID, err)
		}
	}
	if kept.ClosedAt.IsZero() {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO letter_closings (letter_id, closed_at) VALUES ($1, $2)
		ON CONFLICT (letter_id) DO UPDATE SET closed_at = EXCLUDED.closed_at
	`, id, kept.ClosedAt.UTC()); err != nil {
		return fmt.Errorf("failed to upsert the closing of the letter %s: %w", kept.ID, err)
	}
	return nil
}
