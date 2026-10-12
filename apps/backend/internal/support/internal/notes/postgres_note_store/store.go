package postgres_note_store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ notes.Storage = (*Store)(nil)

func (s *Store) Save(ctx context.Context, saved ...notes.Note) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin saving notes: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, note := range saved {
		if err := write(ctx, tx, note.TakeSnapshot()); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the notes saved: %w", err)
	}
	return nil
}

// The keys are the stored format: notes_query reads them in SQL.
type diagnostics struct {
	Build          string `json:"build"`
	ViewportWidth  uint32 `json:"viewportWidth"`
	ViewportHeight uint32 `json:"viewportHeight"`
	ConnectionDown bool   `json:"connectionDown"`
	UserAgent      string `json:"userAgent"`
}

func write(ctx context.Context, tx cppg.Querier, kept notes.Snapshot) error {
	said, err := encoded(kept.Diagnostics)
	if err != nil {
		return fmt.Errorf("failed to encode a note's diagnostics: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO notes (id, kind, account_id, text, diagnostics, submitted_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			kind = EXCLUDED.kind, account_id = EXCLUDED.account_id, text = EXCLUDED.text,
			diagnostics = EXCLUDED.diagnostics, submitted_at = EXCLUDED.submitted_at
	`,
		uuid.UUID(kept.ID), string(kept.Kind), sender(kept.Account), kept.Text, said, kept.At.UTC(),
	); err != nil {
		return fmt.Errorf("failed to upsert a note: %w", err)
	}
	return nil
}

func encoded(said notes.Diagnostics) (sql.NullString, error) {
	if said == (notes.Diagnostics{}) {
		return sql.NullString{}, nil
	}

	payload, err := json.Marshal(diagnostics(said))
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(payload), Valid: true}, nil
}

func decoded(stored sql.NullString) (notes.Diagnostics, error) {
	if !stored.Valid {
		return notes.Diagnostics{}, nil
	}

	var said diagnostics
	if err := json.Unmarshal([]byte(stored.String), &said); err != nil {
		return notes.Diagnostics{}, err
	}
	return notes.Diagnostics(said), nil
}

// An anonymized note has no account: NULL.
func sender(account notes.AccountID) uuid.NullUUID {
	return uuid.NullUUID{UUID: uuid.UUID(account), Valid: account != notes.AccountID{}}
}

func (s *Store) SentBy(ctx context.Context, account notes.AccountID) ([]notes.Note, error) {
	return s.restored(ctx, `
		SELECT id, kind, account_id, text, diagnostics, submitted_at
		FROM notes
		WHERE account_id = $1
		ORDER BY submitted_at, id
	`, uuid.UUID(account))
}

func (s *Store) WithSenderBefore(ctx context.Context, cutoff time.Time, limit int) ([]notes.Note, error) {
	return s.restored(ctx, `
		SELECT id, kind, account_id, text, diagnostics, submitted_at
		FROM notes
		WHERE submitted_at < $1 AND account_id IS NOT NULL
		ORDER BY submitted_at, id
		LIMIT $2
	`, cutoff.UTC(), limit)
}

func (s *Store) restored(ctx context.Context, query string, args ...any) ([]notes.Note, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read notes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var found []notes.Note
	for rows.Next() {
		var (
			id         uuid.UUID
			account    uuid.NullUUID
			kind, text string
			stored     sql.NullString
			at         time.Time
		)
		if err := rows.Scan(&id, &kind, &account, &text, &stored, &at); err != nil {
			return nil, fmt.Errorf("failed to read a note: %w", err)
		}

		said, err := decoded(stored)
		if err != nil {
			return nil, fmt.Errorf("failed to decode the diagnostics of the note %s: %w", id, err)
		}

		note, err := notes.Snapshot{
			ID: notes.NoteID(id), Kind: notes.Kind(kind), Account: notes.AccountID(account.UUID), Text: text,
			Diagnostics: said, At: at.UTC(),
		}.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore the note %s: %w", id, err)
		}
		found = append(found, note)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read notes: %w", err)
	}
	return found, nil
}
