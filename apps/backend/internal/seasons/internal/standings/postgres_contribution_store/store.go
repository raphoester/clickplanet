package postgres_contribution_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ standings.Store = (*Store)(nil)

func (s *Store) Position(ctx context.Context) (standings.Position, error) {
	var position int64
	err := s.db.QueryRowContext(ctx, `SELECT position FROM standings_position`).Scan(&position)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, standings.ErrNotStarted
	}
	if err != nil {
		return 0, fmt.Errorf("failed to read where the standings are counted to: %w", err)
	}
	return standings.Position(position), nil //nolint:gosec // CHECK (position >= 0).
}

func (s *Store) Begin(ctx context.Context, start standings.Position) error {
	begun, err := s.db.ExecContext(ctx,
		`INSERT INTO standings_position (position) VALUES ($1) ON CONFLICT DO NOTHING`, bigint(start))
	if err != nil {
		return fmt.Errorf("failed to save where the standings start: %w", err)
	}
	rows, err := begun.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read whether the standings had started: %w", err)
	}
	if rows == 0 {
		return standings.ErrStarted
	}
	return nil
}

type seasonAccount struct {
	season  calendar.Number
	account standings.AccountID
}

func (s *Store) Count(ctx context.Context, batch standings.Batch, seasons calendar.Calendar) (err error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin counting a batch: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	position, err := lockedPosition(ctx, tx)
	if err != nil {
		return err
	}
	if position != batch.From() {
		return fmt.Errorf("%w: the batch is from %d, the standings are at %d", standings.ErrMoved, batch.From(), position)
	}

	tallies := map[seasonAccount]standings.Tally{}
	var changed []seasonAccount
	for _, take := range batch.Takes() {
		season, ok := take.Season(seasons)
		if !ok {
			continue
		}
		key := seasonAccount{season: season, account: take.Account}
		tally, ok := tallies[key]
		if !ok {
			if tally, err = tallyOf(ctx, tx, season, take.Account); err != nil {
				return err
			}
			changed = append(changed, key)
		}
		tallies[key] = tally.WithTake(take.Country)
	}
	for _, key := range changed {
		if err := saveTally(ctx, tx, key, tallies[key]); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `UPDATE standings_position SET position = $1`, bigint(batch.Next())); err != nil {
		return fmt.Errorf("failed to save where the standings are counted to: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit a batch: %w", err)
	}
	return nil
}

func (s *Store) Rewind(ctx context.Context) (_ standings.Position, err error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to begin a rewind: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := lockedPosition(ctx, tx); err != nil {
		return 0, err
	}
	for _, statement := range []string{`DELETE FROM contributions`, `UPDATE standings_position SET position = 0`} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return 0, fmt.Errorf("failed to put the standings back to the first take: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit a rewind: %w", err)
	}
	return 0, nil
}

func (s *Store) DeleteAccount(ctx context.Context, account standings.AccountID) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM contributions WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to delete the account's contributions: %w", err)
	}
	return nil
}

func lockedPosition(ctx context.Context, tx *sql.Tx) (standings.Position, error) {
	var position int64
	err := tx.QueryRowContext(ctx, `SELECT position FROM standings_position FOR UPDATE`).Scan(&position)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, standings.ErrNotStarted
	}
	if err != nil {
		return 0, fmt.Errorf("failed to lock where the standings are counted to: %w", err)
	}
	return standings.Position(position), nil //nolint:gosec // CHECK (position >= 0).
}

func tallyOf(ctx context.Context, db cppg.Querier, season calendar.Number, account standings.AccountID) (standings.Tally, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT country, tiles, main FROM contributions WHERE season = $1 AND account_id = $2
	`, int64(season), uuid.UUID(account))
	if err != nil {
		return standings.Tally{}, fmt.Errorf("failed to read the account's contributions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tally := standings.Tally{Tiles: map[standings.Country]uint64{}}
	for rows.Next() {
		var (
			country string
			tiles   int64
			main    bool
		)
		if err := rows.Scan(&country, &tiles, &main); err != nil {
			return standings.Tally{}, fmt.Errorf("failed to read a contribution: %w", err)
		}
		tally.Tiles[standings.Country(country)] = uint64(tiles) //nolint:gosec // CHECK (tiles > 0).
		if main {
			tally.Main = standings.Country(country)
		}
	}
	if err := rows.Err(); err != nil {
		return standings.Tally{}, fmt.Errorf("failed to read the account's contributions: %w", err)
	}
	return tally, nil
}

func saveTally(ctx context.Context, tx *sql.Tx, key seasonAccount, tally standings.Tally) error {
	countries := make([]string, 0, len(tally.Tiles))
	tiles := make([]int64, 0, len(tally.Tiles))
	mains := make([]bool, 0, len(tally.Tiles))
	for country, count := range tally.Tiles {
		countries = append(countries, string(country))
		tiles = append(tiles, int64(count)) //nolint:gosec // one a tile taken: never past int64.
		mains = append(mains, country == tally.Main)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM contributions WHERE season = $1 AND account_id = $2`,
		int64(key.season), uuid.UUID(key.account)); err != nil {
		return fmt.Errorf("failed to clear the account's contributions: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO contributions (season, account_id, country, tiles, main)
		SELECT $1, $2, country, tiles, main FROM unnest($3::text[], $4::bigint[], $5::boolean[]) AS counted (country, tiles, main)
	`, int64(key.season), uuid.UUID(key.account), pq.Array(countries), pq.Array(tiles), pq.Array(mains)); err != nil {
		return fmt.Errorf("failed to save the account's contributions: %w", err)
	}
	return nil
}

func bigint(position standings.Position) int64 {
	return int64(position) //nolint:gosec // a position comes from a bigint.
}
