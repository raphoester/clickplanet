package postgres_moderation_store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ moderation.Storage = (*Store)(nil)

func (s *Store) Record(ctx context.Context, action moderation.Action) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO moderation (account_id, action, acted_at, mute_kind, seconds, message_id, text, gif_url, gif_width,
			gif_height, reason, operator_id, client)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (message_id) WHERE action = 'removal' DO NOTHING
	`, row(action.TakeSnapshot())...); err != nil {
		return fmt.Errorf("failed to record a moderation action: %w", err)
	}
	return nil
}

func (s *Store) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM moderation WHERE acted_at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to prune the moderation history: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count the moderation actions pruned: %w", err)
	}
	return deleted, nil
}

func (s *Store) Forget(ctx context.Context, account messages.AccountID) error {
	return s.db.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // each statement names what failed.
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `DELETE FROM moderation WHERE account_id = $1`, uuid.UUID(account)); err != nil {
			return fmt.Errorf("failed to forget the actions taken on an account: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE moderation SET operator_id = NULL WHERE operator_id = $1
		`, uuid.UUID(account)); err != nil {
			return fmt.Errorf("failed to take an operator off its actions: %w", err)
		}
		return nil
	})
}

func row(action moderation.Snapshot) []any {
	var (
		muteKind sql.NullString
		seconds  sql.NullInt64
		message  sql.NullString
		text     sql.NullString
		gifURL   sql.NullString
		width    sql.NullInt64
		height   sql.NullInt64
	)
	switch action.Kind {
	case moderation.Muted:
		muteKind = sql.NullString{String: action.MuteKind.String(), Valid: true}
		seconds = sql.NullInt64{Int64: int64(action.Duration / time.Second), Valid: true}
	case moderation.Unmuted:
		muteKind = sql.NullString{String: action.MuteKind.String(), Valid: true}
	case moderation.Removed:
		message = sql.NullString{String: string(action.Message), Valid: true}
		text = sql.NullString{String: action.Text, Valid: true}
		if !action.Gif.None() {
			gifURL = sql.NullString{String: action.Gif.URL, Valid: true}
			width = sql.NullInt64{Int64: int64(action.Gif.Width), Valid: true}
			height = sql.NullInt64{Int64: int64(action.Gif.Height), Valid: true}
		}
	}

	return []any{
		account(action.Account),
		action.Kind.String(),
		action.At.UTC(),
		muteKind,
		seconds,
		message,
		text,
		gifURL,
		width,
		height,
		string(action.Reason),
		account(action.By.Account()),
		action.By.Client(),
	}
}

func account(id messages.AccountID) uuid.NullUUID {
	if id == messages.NoAccount {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: uuid.UUID(id), Valid: true}
}
