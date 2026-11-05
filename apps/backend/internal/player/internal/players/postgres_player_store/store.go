package postgres_player_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ players.Store = (*Store)(nil)

func (s *Store) Profile(ctx context.Context, account players.AccountID) (players.Profile, error) {
	var (
		name      string
		updatedAt time.Time
		chosenAt  sql.NullTime
		admin     bool
		color     int32
	)
	err := s.db.On(ctx).QueryRowContext(
		ctx,
		`SELECT name, updated_at, name_chosen_at, admin, color FROM profiles WHERE account_id = $1`, uuid.UUID(account),
	).Scan(&name, &updatedAt, &chosenAt, &admin, &color)
	if errors.Is(err, sql.ErrNoRows) {
		return players.Profile{}, players.ErrNoProfile
	}
	if err != nil {
		return players.Profile{}, fmt.Errorf("failed to read the profile: %w", err)
	}

	snapshot := players.ProfileSnapshot{
		Account: account, Name: players.Name(name), UpdatedAt: updatedAt.UTC(), Admin: admin, Color: players.Color(color),
	}
	if chosenAt.Valid {
		snapshot.ChosenAt = chosenAt.Time.UTC()
	}
	return snapshot.Restore(), nil
}

const uniqueNameIndex = "profiles_name_key"

const uniqueViolation = "23505"

func (s *Store) SaveProfile(ctx context.Context, profile players.Profile) error {
	kept := profile.TakeSnapshot()
	_, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO profiles (account_id, name, name_folded, updated_at, name_chosen_at) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (account_id) DO UPDATE SET
			name = excluded.name, name_folded = excluded.name_folded, updated_at = excluded.updated_at,
			name_chosen_at = excluded.name_chosen_at
	`, uuid.UUID(kept.Account), string(kept.Name), kept.Name.Folded(), kept.UpdatedAt.UTC(), nullableTime(kept.ChosenAt))

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == uniqueViolation && pqErr.Constraint == uniqueNameIndex {
		return players.ErrNameTaken
	}
	if err != nil {
		return fmt.Errorf("failed to save the profile: %w", err)
	}
	return nil
}

// A taken name raises no error, so the transaction it runs in, which records the player made, goes on to the next draw.
func (s *Store) CreateProfile(ctx context.Context, profile players.Profile) error {
	kept := profile.TakeSnapshot()
	result, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO profiles (account_id, name, name_folded, updated_at, name_chosen_at) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING
	`, uuid.UUID(kept.Account), string(kept.Name), kept.Name.Folded(), kept.UpdatedAt.UTC(), nullableTime(kept.ChosenAt))
	if err != nil {
		return fmt.Errorf("failed to create the profile: %w", err)
	}
	created, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to count the profiles created: %w", err)
	}
	if created == 1 {
		return nil
	}

	var exists bool
	if err := s.db.On(ctx).QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM profiles WHERE account_id = $1)`,
		uuid.UUID(kept.Account)).Scan(&exists); err != nil {
		return fmt.Errorf("failed to read whether the account has a profile: %w", err)
	}
	if exists {
		return players.ErrProfileExists
	}
	return players.ErrNameTaken
}

func (s *Store) SaveColor(ctx context.Context, account players.AccountID, color players.Color) error {
	result, err := s.db.On(ctx).ExecContext(ctx, `UPDATE profiles SET color = $2 WHERE account_id = $1`, uuid.UUID(account), int32(color))
	if err != nil {
		return fmt.Errorf("failed to save the color: %w", err)
	}
	saved, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to count the colors saved: %w", err)
	}
	if saved == 0 {
		return players.ErrNoProfile
	}
	return nil
}

func (s *Store) GuestCode(ctx context.Context, account players.AccountID) (players.GuestCode, error) {
	var code string
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT code FROM guest_codes WHERE account_id = $1`, uuid.UUID(account)).Scan(&code)
	if errors.Is(err, sql.ErrNoRows) {
		return "", players.ErrNoGuestCode
	}
	if err != nil {
		return "", fmt.Errorf("failed to read the guest code: %w", err)
	}
	return players.GuestCode(code), nil
}

const uniqueGuestCode = "guest_codes_code_key"

func (s *Store) SaveGuestCode(ctx context.Context, account players.AccountID, code players.GuestCode) error {
	_, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO guest_codes (account_id, code) VALUES ($1, $2)
		ON CONFLICT (account_id) DO NOTHING
	`, uuid.UUID(account), string(code))

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == uniqueViolation && pqErr.Constraint == uniqueGuestCode {
		return players.ErrGuestCodeTaken
	}
	if err != nil {
		return fmt.Errorf("failed to save the guest code: %w", err)
	}
	return nil
}

func (s *Store) DeleteAccount(ctx context.Context, account players.AccountID) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		for _, statement := range []string{
			`DELETE FROM profiles WHERE account_id = $1`,
			`DELETE FROM guest_codes WHERE account_id = $1`,
		} {
			if _, err := tx.ExecContext(ctx, statement, uuid.UUID(account)); err != nil {
				return fmt.Errorf("failed to delete the account's rows: %w", err)
			}
		}
		return nil
	})
}

func nullableTime(at time.Time) sql.NullTime {
	return sql.NullTime{Time: at.UTC(), Valid: !at.IsZero()}
}
