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

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ mutes.Storage = (*Store)(nil)

func (s *Store) Save(ctx context.Context, mute mutes.Mute) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO mutes (id, account_id, scope, muted_at, muted_until)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			account_id = EXCLUDED.account_id, scope = EXCLUDED.scope,
			muted_at = EXCLUDED.muted_at, muted_until = EXCLUDED.muted_until
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
	err := s.db.On(ctx).QueryRowContext(ctx, `
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

	return muteOf(id, account, kept, mutedAt, until), nil
}

func (s *Store) MutesOf(ctx context.Context, account messages.AccountID) ([]mutes.Mute, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT id, account_id, scope, muted_at, muted_until
		FROM mutes
		WHERE account_id = $1
	`, uuid.UUID(account))
	if err != nil {
		return nil, fmt.Errorf("failed to read the account's mutes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var found []mutes.Mute
	for rows.Next() {
		var (
			id      uuid.UUID
			owner   uuid.UUID
			kept    sql.NullString
			mutedAt time.Time
			until   time.Time
		)
		if err := rows.Scan(&id, &owner, &kept, &mutedAt, &until); err != nil {
			return nil, fmt.Errorf("failed to scan a mute: %w", err)
		}
		found = append(found, muteOf(id, owner, kept, mutedAt, until))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the account's mutes: %w", err)
	}
	return found, nil
}

func muteOf(id uuid.UUID, account uuid.UUID, scope sql.NullString, at time.Time, until time.Time) mutes.Mute {
	return mutes.MuteOf(mutes.MuteID(id), mutes.CallerOf(messages.AccountID(account), mutes.Scope(scope.String)),
		at.UTC(), until.UTC())
}

func (s *Store) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM mutes WHERE muted_until < $1`, cutoff)
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
