package postgres_mute_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ mutes.Storage = (*Store)(nil)

func (s *Store) Save(ctx context.Context, mute mutes.Mute) error {
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO mutes (id, account_id, scope, muted_at, muted_until)
		VALUES ($1, $2, $3, $4, $5)
	`,
		uuid.UUID(mute.ID()), uuid.UUID(mute.Caller().Account()), scope(mute.Caller().Scope()),
		mute.At().UTC(), mute.Until().UTC(),
	); err != nil {
		return fmt.Errorf("failed to insert a mute: %w", err)
	}
	return nil
}

func (s *Store) Mute(ctx context.Context, caller mutes.Caller, at time.Time) (mutes.Mute, error) {
	var (
		id      uuid.UUID
		account uuid.UUID
		kept    sql.NullString
		mutedAt time.Time
		until   time.Time
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT id, account_id, scope, muted_at, muted_until
		FROM mutes
		WHERE muted_until > $3 AND (account_id = $1 OR scope = $2)
		ORDER BY muted_until DESC, id
		LIMIT 1
	`, uuid.UUID(caller.Account()), scope(caller.Scope()), at.UTC()).Scan(&id, &account, &kept, &mutedAt, &until)
	if errors.Is(err, sql.ErrNoRows) {
		return mutes.Mute{}, mutes.ErrNotMuted
	}
	if err != nil {
		return mutes.Mute{}, fmt.Errorf("failed to read a mute: %w", err)
	}

	return mutes.MuteOf(mutes.MuteID(id), mutes.CallerOf(messages.AccountID(account), mutes.Scope(kept.String)),
		mutedAt.UTC(), until.UTC()), nil
}

func (s *Store) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM mutes WHERE muted_until < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old mutes: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count deleted mutes: %w", err)
	}
	return deleted, nil
}

func scope(scope mutes.Scope) sql.NullString {
	return sql.NullString{String: string(scope), Valid: scope != mutes.NoScope}
}
