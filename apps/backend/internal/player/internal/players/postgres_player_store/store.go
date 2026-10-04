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
		color     int32
	)
	err := s.db.QueryRowContext(ctx, `SELECT name, updated_at, admin, color FROM profiles WHERE account_id = $1`, uuid.UUID(account)).
		Scan(&name, &updatedAt, &admin, &color)
	if errors.Is(err, sql.ErrNoRows) {
		return players.Profile{}, players.ErrNoProfile
	}
	if err != nil {
		return players.Profile{}, fmt.Errorf("failed to read the profile: %w", err)
	}
	return players.ProfileOf(account, players.Name(name), updatedAt.UTC(), admin, players.Color(color)), nil
}

const uniqueNameIndex = "profiles_name_key"

const uniqueViolation = "23505"

func (s *Store) SaveProfile(ctx context.Context, profile players.Profile) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO profiles (account_id, name, name_folded, updated_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT (account_id) DO UPDATE SET
			name = excluded.name, name_folded = excluded.name_folded, updated_at = excluded.updated_at
	`, uuid.UUID(profile.Account()), string(profile.Name()), profile.Name().Folded(), profile.UpdatedAt().UTC())

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == uniqueViolation && pqErr.Constraint == uniqueNameIndex {
		return players.ErrNameTaken
	}
	if err != nil {
		return fmt.Errorf("failed to save the profile: %w", err)
	}
	return nil
}

func (s *Store) CreateProfile(ctx context.Context, profile players.Profile) error {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO profiles (account_id, name, name_folded, updated_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT (account_id) DO NOTHING
	`, uuid.UUID(profile.Account()), string(profile.Name()), profile.Name().Folded(), profile.UpdatedAt().UTC())

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == uniqueViolation && pqErr.Constraint == uniqueNameIndex {
		return players.ErrNameTaken
	}
	if err != nil {
		return fmt.Errorf("failed to create the profile: %w", err)
	}
	created, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to count the profiles created: %w", err)
	}
	if created == 0 {
		return players.ErrProfileExists
	}
	return nil
}

func (s *Store) SaveColor(ctx context.Context, account players.AccountID, color players.Color) error {
	result, err := s.db.ExecContext(ctx, `UPDATE profiles SET color = $2 WHERE account_id = $1`, uuid.UUID(account), int32(color))
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
	err := s.db.QueryRowContext(ctx, `SELECT code FROM guest_codes WHERE account_id = $1`, uuid.UUID(account)).Scan(&code)
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
	_, err := s.db.ExecContext(ctx, `
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

const statsColumns = `account_id, tiles_taken, streak_current, streak_best, streak_last_day, messages_sent`

func (s *Store) Stats(ctx context.Context, account players.AccountID) (players.Stats, error) {
	return statsOf(s.db.QueryRowContext(ctx, `SELECT `+statsColumns+` FROM stats WHERE account_id = $1`, uuid.UUID(account)))
}

const statsLock = 0x706c6179

const upsertStats = `INSERT INTO stats (` + statsColumns + `) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (account_id) DO UPDATE SET `

// The takes store writes the take columns in its own transaction: a message never writes them.
const upsertMessages = upsertStats + `messages_sent = excluded.messages_sent`

func (s *Store) RecordMessage(ctx context.Context, account players.AccountID) error {
	return s.record(ctx, account, players.Stats.WithMessage, upsertMessages)
}

func (s *Store) record(ctx context.Context, account players.AccountID, change func(players.Stats) players.Stats, upsert string) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin changing the stats: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, statsLock, account.String()); err != nil {
		return fmt.Errorf("failed to lock the account's stats: %w", err)
	}

	current, err := statsOf(tx.QueryRowContext(ctx, `SELECT `+statsColumns+` FROM stats WHERE account_id = $1`, uuid.UUID(account)))
	if errors.Is(err, players.ErrNoStats) {
		current = players.NewStats(account)
		err = nil
	}
	if err != nil {
		return err
	}

	next := change(current)
	if _, err := tx.ExecContext(ctx, upsert, uuid.UUID(account), int64(next.TilesTaken()), //nolint:gosec // one a tile taken: never past int64.
		int64(next.Streak().Days()), int64(next.StreakBest()), nullableDay(next.Streak().LastDay()),
		int64(next.MessagesSent())); err != nil { //nolint:gosec // one a message sent: never past int64.
		return fmt.Errorf("failed to save the stats: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the stats: %w", err)
	}
	return nil
}

func (s *Store) StatsAfter(ctx context.Context, after players.AccountID, limit int) ([]players.Stats, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+statsColumns+` FROM stats
		WHERE account_id > $1 ORDER BY account_id LIMIT $2
	`, uuid.UUID(after), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to read a page of stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var page []players.Stats
	for rows.Next() {
		stats, err := statsOf(rows)
		if err != nil {
			return nil, err
		}
		page = append(page, stats)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read a page of stats: %w", err)
	}
	return page, nil
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
		`DELETE FROM guest_codes WHERE account_id = $1`,
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

type scanner interface {
	Scan(dest ...any) error
}

func statsOf(row scanner) (players.Stats, error) {
	var (
		account                        uuid.UUID
		tiles, current, best, messages int64
		lastDay                        sql.NullTime
	)
	err := row.Scan(&account, &tiles, &current, &best, &lastDay, &messages)
	if errors.Is(err, sql.ErrNoRows) {
		return players.Stats{}, players.ErrNoStats
	}
	if err != nil {
		return players.Stats{}, fmt.Errorf("failed to read the stats: %w", err)
	}

	var day players.Day
	if lastDay.Valid {
		day = players.DayOf(lastDay.Time)
	}
	return players.StatsOf(
		players.AccountID(account),
		uint64(tiles),                          //nolint:gosec // CHECK (tiles_taken >= 0).
		players.StreakOf(uint32(current), day), //nolint:gosec // CHECK (streak_current >= 0), and one a day.
		uint32(best),                           //nolint:gosec // as above.
		uint64(messages),                       //nolint:gosec // CHECK (messages_sent >= 0).
	), nil
}

func nullableDay(day players.Day) sql.NullString {
	return sql.NullString{String: day.String(), Valid: !day.Empty()}
}
