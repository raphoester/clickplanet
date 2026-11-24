package postgres_catalog_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const heardLock = 0x666c6167

// Store keeps each flag heard as one row of planet.heard_flags.
func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ catalog.Store = (*Store)(nil)

const columns = `flag, state::text, decided_at, pin_latitude, pin_longitude, pinned_at`

func (s *Store) Heard(ctx context.Context, flag string) (catalog.Heard, error) {
	kept, err := keptOf(ctx, s.db.On(ctx), flag)
	if err != nil {
		return catalog.Heard{}, err
	}
	heard, err := kept.Restore()
	if err != nil {
		return catalog.Heard{}, fmt.Errorf("failed to restore the flag heard: %w", err)
	}
	return heard, nil
}

func (s *Store) All(ctx context.Context) (map[string]catalog.Heard, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT `+columns+` FROM heard_flags`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the flags heard: %w", err)
	}
	defer func() { _ = rows.Close() }()

	all := map[string]catalog.Heard{}
	for rows.Next() {
		flag, kept, err := scan(rows)
		if err != nil {
			return nil, err
		}
		heard, err := kept.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore the flag heard %q: %w", flag, err)
		}
		all[flag] = heard
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the flags heard: %w", err)
	}
	return all, nil
}

func (s *Store) Change(ctx context.Context, flag string, change func(catalog.Heard) catalog.Heard) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, heardLock, flag); err != nil {
			return fmt.Errorf("failed to lock the flag heard: %w", err)
		}
		before, err := keptOf(ctx, tx, flag)
		if err != nil {
			return err
		}
		heard, err := before.Restore()
		if err != nil {
			return fmt.Errorf("failed to restore the flag heard: %w", err)
		}
		after := change(heard).TakeSnapshot()
		if after == before {
			return nil
		}
		latitude, longitude, pinnedAt := pinOf(after)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO heard_flags (flag, state, decided_at, pin_latitude, pin_longitude, pinned_at)
			VALUES ($1, $2::heard_flag_state, $3, $4, $5, $6)
			ON CONFLICT (flag) DO UPDATE SET state = EXCLUDED.state, decided_at = EXCLUDED.decided_at,
				pin_latitude = EXCLUDED.pin_latitude, pin_longitude = EXCLUDED.pin_longitude, pinned_at = EXCLUDED.pinned_at
		`, flag, string(after.State), after.DecidedAt.UTC(), latitude, longitude, pinnedAt); err != nil {
			return fmt.Errorf("failed to keep the flag heard: %w", err)
		}
		return nil
	})
}

// A flag no fact gave a pin keeps none: NULL, never 0, 0.
func pinOf(kept catalog.HeardSnapshot) (sql.NullFloat64, sql.NullFloat64, sql.NullTime) {
	if kept.PinnedAt.IsZero() {
		return sql.NullFloat64{}, sql.NullFloat64{}, sql.NullTime{}
	}
	return sql.NullFloat64{Float64: kept.Pin.Latitude, Valid: true}, sql.NullFloat64{Float64: kept.Pin.Longitude, Valid: true},
		sql.NullTime{Time: kept.PinnedAt.UTC(), Valid: true}
}

func keptOf(ctx context.Context, db cppg.Querier, flag string) (catalog.HeardSnapshot, error) {
	_, kept, err := scan(db.QueryRowContext(ctx, `SELECT `+columns+` FROM heard_flags WHERE flag = $1`, flag))
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.HeardSnapshot{}, nil
	}
	return kept, err
}

type scanner interface {
	Scan(dest ...any) error
}

func scan(row scanner) (string, catalog.HeardSnapshot, error) {
	var (
		flag, state         string
		decidedAt           time.Time
		latitude, longitude sql.NullFloat64
		pinnedAt            sql.NullTime
	)
	if err := row.Scan(&flag, &state, &decidedAt, &latitude, &longitude, &pinnedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", catalog.HeardSnapshot{}, err //nolint:wrapcheck // the sentinel, unwrapped: no row is no flag heard.
		}
		return "", catalog.HeardSnapshot{}, fmt.Errorf("failed to read the flag heard: %w", err)
	}
	kept := catalog.HeardSnapshot{State: catalog.State(state), DecidedAt: decidedAt.UTC()}
	if pinnedAt.Valid {
		kept.Pin = clicks.Pin{Latitude: latitude.Float64, Longitude: longitude.Float64}
		kept.PinnedAt = pinnedAt.Time.UTC()
	}
	return flag, kept, nil
}
