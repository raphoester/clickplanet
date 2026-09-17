// Package postgres_player_store keeps the in-memory players between boots, in player.profiles and player.stats.
package postgres_player_store

import (
	"context"
	"database/sql"
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

var _ players.Persistence = (*Store)(nil)

func (s *Store) Load(ctx context.Context) (players.Snapshot, error) {
	profiles, err := s.loadProfiles(ctx)
	if err != nil {
		return players.Snapshot{}, err
	}
	stats, err := s.loadStats(ctx)
	if err != nil {
		return players.Snapshot{}, err
	}
	return players.Snapshot{Profiles: profiles, Stats: stats}, nil
}

func (s *Store) loadProfiles(ctx context.Context) ([]players.Profile, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT account_id, name, updated_at FROM profiles`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the profiles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var profiles []players.Profile
	for rows.Next() {
		var (
			account   uuid.UUID
			name      string
			updatedAt time.Time
		)
		if err := rows.Scan(&account, &name, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to read a profile: %w", err)
		}
		profiles = append(profiles, players.Profile{
			Account: players.AccountID(account), Name: players.Name(name), UpdatedAt: updatedAt.UTC(),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the profiles: %w", err)
	}
	return profiles, nil
}

func (s *Store) loadStats(ctx context.Context) ([]players.Stats, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT account_id, tiles_taken, streak_current, streak_best, streak_last_day FROM stats
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var all []players.Stats
	for rows.Next() {
		var (
			account              uuid.UUID
			tiles, current, best int64
			lastDay              time.Time
		)
		if err := rows.Scan(&account, &tiles, &current, &best, &lastDay); err != nil {
			return nil, fmt.Errorf("failed to read a stats row: %w", err)
		}
		all = append(all, players.Stats{
			Account:       players.AccountID(account),
			TilesTaken:    uint64(tiles),   //nolint:gosec // CHECK (tiles_taken >= 0).
			StreakCurrent: uint32(current), //nolint:gosec // CHECK (streak_current >= 0), and one a day.
			StreakBest:    uint32(best),    //nolint:gosec // as above.
			StreakLastDay: players.DayOf(lastDay),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the stats: %w", err)
	}
	return all, nil
}

// Save upserts the rows and deletes the others, a statement per kind, in one transaction.
func (s *Store) Save(ctx context.Context, changes players.Changes) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin saving the players: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for _, write := range []func(context.Context, *sql.Tx, players.Changes) error{
		saveProfiles, saveStats, deleteProfiles, deleteStats,
	} {
		if err := write(ctx, tx, changes); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the players: %w", err)
	}
	return nil
}

func saveProfiles(ctx context.Context, tx *sql.Tx, changes players.Changes) error {
	if len(changes.Profiles) == 0 {
		return nil
	}

	accounts := make([]string, len(changes.Profiles))
	names := make([]string, len(changes.Profiles))
	updatedAt := make([]time.Time, len(changes.Profiles))
	for i, profile := range changes.Profiles {
		accounts[i], names[i], updatedAt[i] = profile.Account.String(), string(profile.Name), profile.UpdatedAt.UTC()
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO profiles (account_id, name, updated_at)
		SELECT * FROM unnest($1::uuid[], $2::text[], $3::timestamptz[])
		ON CONFLICT (account_id) DO UPDATE SET name = excluded.name, updated_at = excluded.updated_at
	`, pq.Array(accounts), pq.Array(names), pq.Array(updatedAt)); err != nil {
		return fmt.Errorf("failed to save the profiles: %w", err)
	}
	return nil
}

func saveStats(ctx context.Context, tx *sql.Tx, changes players.Changes) error {
	if len(changes.Stats) == 0 {
		return nil
	}

	n := len(changes.Stats)
	accounts, lastDays := make([]string, n), make([]string, n)
	tiles, current, best := make([]int64, n), make([]int64, n), make([]int64, n)
	for i, stats := range changes.Stats {
		accounts[i], lastDays[i] = stats.Account.String(), stats.StreakLastDay.String()
		tiles[i] = int64(stats.TilesTaken) //nolint:gosec // one a tile taken: never past int64.
		current[i], best[i] = int64(stats.StreakCurrent), int64(stats.StreakBest)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO stats (account_id, tiles_taken, streak_current, streak_best, streak_last_day)
		SELECT * FROM unnest($1::uuid[], $2::bigint[], $3::bigint[], $4::bigint[], $5::date[])
		ON CONFLICT (account_id) DO UPDATE SET
			tiles_taken = excluded.tiles_taken,
			streak_current = excluded.streak_current,
			streak_best = excluded.streak_best,
			streak_last_day = excluded.streak_last_day
	`, pq.Array(accounts), pq.Array(tiles), pq.Array(current), pq.Array(best), pq.Array(lastDays)); err != nil {
		return fmt.Errorf("failed to save the stats: %w", err)
	}
	return nil
}

func deleteProfiles(ctx context.Context, tx *sql.Tx, changes players.Changes) error {
	return deleteRows(ctx, tx, `DELETE FROM profiles WHERE account_id = ANY($1::uuid[])`, changes.DeletedProfiles)
}

func deleteStats(ctx context.Context, tx *sql.Tx, changes players.Changes) error {
	return deleteRows(ctx, tx, `DELETE FROM stats WHERE account_id = ANY($1::uuid[])`, changes.DeletedStats)
}

func deleteRows(ctx context.Context, tx *sql.Tx, statement string, deleted []players.AccountID) error {
	if len(deleted) == 0 {
		return nil
	}

	accounts := make([]string, len(deleted))
	for i, account := range deleted {
		accounts[i] = account.String()
	}

	if _, err := tx.ExecContext(ctx, statement, pq.Array(accounts)); err != nil {
		return fmt.Errorf("failed to delete the players' rows: %w", err)
	}
	return nil
}
