// Package postgres_player_store is players.Store over player.profiles and player.stats.
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

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ players.Store = (*Store)(nil)

func (s *Store) Profile(ctx context.Context, account players.AccountID) (players.Profile, error) {
	var (
		name      string
		updatedAt time.Time
		admin     bool
	)
	err := s.db.QueryRowContext(ctx, `SELECT name, updated_at, admin FROM profiles WHERE account_id = $1`, uuid.UUID(account)).
		Scan(&name, &updatedAt, &admin)
	if errors.Is(err, sql.ErrNoRows) {
		return players.Profile{}, players.ErrNoProfile
	}
	if err != nil {
		return players.Profile{}, fmt.Errorf("failed to read the profile: %w", err)
	}
	return players.Profile{Account: account, Name: players.Name(name), UpdatedAt: updatedAt.UTC(), Admin: admin}, nil
}

// ProfileNamed reads through the unique index on lower(name).
func (s *Store) ProfileNamed(ctx context.Context, name players.Name) (players.Profile, error) {
	var (
		account   uuid.UUID
		held      string
		updatedAt time.Time
		admin     bool
	)
	err := s.db.QueryRowContext(ctx, `SELECT account_id, name, updated_at, admin FROM profiles WHERE lower(name) = $1`, name.Folded()).
		Scan(&account, &held, &updatedAt, &admin)
	if errors.Is(err, sql.ErrNoRows) {
		return players.Profile{}, players.ErrNoProfile
	}
	if err != nil {
		return players.Profile{}, fmt.Errorf("failed to read the profile by name: %w", err)
	}
	return players.Profile{
		Account: players.AccountID(account), Name: players.Name(held), UpdatedAt: updatedAt.UTC(), Admin: admin,
	}, nil
}

// uniqueNameIndex is the unique index on lower(name), which a name another account holds violates.
const uniqueNameIndex = "profiles_name_key"

// uniqueViolation is postgres' unique_violation.
const uniqueViolation = "23505"

// SaveProfile leaves uniqueness to the index, so two players asking for one name at once cannot both get it.
func (s *Store) SaveProfile(ctx context.Context, profile players.Profile) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO profiles (account_id, name, updated_at) VALUES ($1, $2, $3)
		ON CONFLICT (account_id) DO UPDATE SET name = excluded.name, updated_at = excluded.updated_at
	`, uuid.UUID(profile.Account), string(profile.Name), profile.UpdatedAt.UTC())

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == uniqueViolation && pqErr.Constraint == uniqueNameIndex {
		return players.ErrNameTaken
	}
	if err != nil {
		return fmt.Errorf("failed to save the profile: %w", err)
	}
	return nil
}

func (s *Store) Stats(ctx context.Context, account players.AccountID) (players.Stats, error) {
	return statsOf(s.db.QueryRowContext(ctx, `
		SELECT tiles_taken, streak_current, streak_best, streak_last_day FROM stats WHERE account_id = $1
	`, uuid.UUID(account)), account)
}

// takesLock is the advisory lock space of RecordTake, so its keys meet no other lock in the database.
const takesLock = 0x706c6179 // "play"

// RecordTake holds a lock on the account for the transaction while the domain's rule computes the next
// stats, so two takes never read the same stats. An advisory lock rather than FOR UPDATE: a first take has
// no row to lock yet.
func (s *Store) RecordTake(ctx context.Context, account players.AccountID, at time.Time) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin recording a take: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, takesLock, account.String()); err != nil {
		return fmt.Errorf("failed to lock the account's stats: %w", err)
	}

	current, err := statsOf(tx.QueryRowContext(ctx, `
		SELECT tiles_taken, streak_current, streak_best, streak_last_day FROM stats WHERE account_id = $1
	`, uuid.UUID(account)), account)
	if errors.Is(err, players.ErrNoStats) {
		current, err = players.Stats{Account: account}, nil
	}
	if err != nil {
		return err
	}

	next := current.WithTake(at)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO stats (account_id, tiles_taken, streak_current, streak_best, streak_last_day)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (account_id) DO UPDATE SET
			tiles_taken = excluded.tiles_taken,
			streak_current = excluded.streak_current,
			streak_best = excluded.streak_best,
			streak_last_day = excluded.streak_last_day
	`, uuid.UUID(account), int64(next.TilesTaken), //nolint:gosec // one a tile taken: never past int64.
		int64(next.StreakCurrent), int64(next.StreakBest), next.StreakLastDay.String()); err != nil {
		return fmt.Errorf("failed to save the stats: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the take: %w", err)
	}
	return nil
}

func (s *Store) DeleteAccount(ctx context.Context, account players.AccountID) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin deleting the account: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for _, statement := range []string{
		`DELETE FROM profiles WHERE account_id = $1`,
		`DELETE FROM stats WHERE account_id = $1`,
	} {
		if _, err := tx.ExecContext(ctx, statement, uuid.UUID(account)); err != nil {
			return fmt.Errorf("failed to delete the account's rows: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the deletion: %w", err)
	}
	return nil
}

func (s *Store) Names(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]players.Name, error) {
	names := make(map[players.AccountID]players.Name, len(accounts))
	if len(accounts) == 0 {
		return names, nil
	}

	// lib/pq cannot take a named array, so the ids go as text.
	ids := make([]string, len(accounts))
	for i, account := range accounts {
		ids[i] = account.String()
	}

	rows, err := s.db.QueryContext(ctx, `SELECT account_id, name FROM profiles WHERE account_id = ANY($1::uuid[])`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to read the names: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			account uuid.UUID
			name    string
		)
		if err := rows.Scan(&account, &name); err != nil {
			return nil, fmt.Errorf("failed to read a name: %w", err)
		}
		names[players.AccountID(account)] = players.Name(name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the names: %w", err)
	}
	return names, nil
}

// statsOf reads one stats row, or ErrNoStats.
func statsOf(row *sql.Row, account players.AccountID) (players.Stats, error) {
	var (
		tiles, current, best int64
		lastDay              time.Time
	)
	err := row.Scan(&tiles, &current, &best, &lastDay)
	if errors.Is(err, sql.ErrNoRows) {
		return players.Stats{}, players.ErrNoStats
	}
	if err != nil {
		return players.Stats{}, fmt.Errorf("failed to read the stats: %w", err)
	}

	return players.Stats{
		Account:       account,
		TilesTaken:    uint64(tiles),   //nolint:gosec // CHECK (tiles_taken >= 0).
		StreakCurrent: uint32(current), //nolint:gosec // CHECK (streak_current >= 0), and one a day.
		StreakBest:    uint32(best),    //nolint:gosec // as above.
		StreakLastDay: players.DayOf(lastDay),
	}, nil
}
