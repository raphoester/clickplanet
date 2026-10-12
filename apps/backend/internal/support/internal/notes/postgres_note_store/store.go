package postgres_note_store

import (
	"context"
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

func write(ctx context.Context, tx cppg.Querier, kept notes.Snapshot) error {
	if kept.Forgotten {
		if _, err := tx.ExecContext(ctx, `DELETE FROM notes WHERE id = $1`, uuid.UUID(kept.ID)); err != nil {
			return fmt.Errorf("failed to delete a forgotten note: %w", err)
		}
		return nil
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO notes (id, kind, account_id, text, submitted_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			kind = EXCLUDED.kind, account_id = EXCLUDED.account_id, text = EXCLUDED.text, submitted_at = EXCLUDED.submitted_at
	`,
		uuid.UUID(kept.ID), string(kept.Kind), uuid.UUID(kept.Account), kept.Text, kept.At.UTC(),
	); err != nil {
		return fmt.Errorf("failed to upsert a note: %w", err)
	}
	return nil
}

func (s *Store) SentBy(ctx context.Context, account notes.AccountID) ([]notes.Note, error) {
	return s.restored(ctx, `
		SELECT id, kind, account_id, text, submitted_at
		FROM notes
		WHERE account_id = $1
		ORDER BY submitted_at, id
	`, uuid.UUID(account))
}

func (s *Store) SentBefore(ctx context.Context, cutoff time.Time, limit int) ([]notes.Note, error) {
	return s.restored(ctx, `
		SELECT id, kind, account_id, text, submitted_at
		FROM notes
		WHERE submitted_at < $1
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
			id, account uuid.UUID
			kind, text  string
			at          time.Time
		)
		if err := rows.Scan(&id, &kind, &account, &text, &at); err != nil {
			return nil, fmt.Errorf("failed to read a note: %w", err)
		}

		note, err := notes.Snapshot{
			ID: notes.NoteID(id), Kind: notes.Kind(kind), Account: notes.AccountID(account), Text: text, At: at.UTC(),
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
