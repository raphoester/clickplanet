package postgres_contribution_store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

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

const takeLock = 0x73656173

func (s *Store) RecordTake(ctx context.Context, season calendar.Number, take standings.Take) (err error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin counting the take: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, takeLock, take.Account.String()); err != nil {
		return fmt.Errorf("failed to lock the account's contributions: %w", err)
	}

	tally, err := tallyOf(ctx, tx, season, take.Account)
	if err != nil {
		return err
	}
	next := tally.WithTake(take.Country)

	if tally.Main != "" && next.Main != tally.Main {
		if _, err := tx.ExecContext(ctx, `
			UPDATE contributions SET main = false WHERE season = $1 AND account_id = $2 AND country = $3
		`, int64(season), uuid.UUID(take.Account), string(tally.Main)); err != nil {
			return fmt.Errorf("failed to move the main flag: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO contributions (season, account_id, country, tiles, main) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (season, account_id, country) DO UPDATE SET tiles = excluded.tiles, main = excluded.main
	`, int64(season), uuid.UUID(take.Account), string(take.Country),
		int64(next.Tiles[take.Country]), next.Main == take.Country); err != nil { //nolint:gosec // one a tile taken: never past int64.
		return fmt.Errorf("failed to save the take: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the take: %w", err)
	}
	return nil
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

func (s *Store) Line(ctx context.Context, season calendar.Number, account standings.AccountID) (standings.Line, error) {
	tally, err := tallyOf(ctx, s.db, season, account)
	if err != nil {
		return standings.Line{}, err
	}
	if tally.Empty() {
		return standings.Line{}, standings.ErrNoLine
	}
	return tally.LineOf(account), nil
}

const linesColumns = `SELECT account_id, country, tiles FROM contributions`

const linesFrom = `
	AND (tiles < $2 OR (tiles = $2 AND account_id > $3))
	ORDER BY tiles DESC, account_id
	LIMIT $4
`

func (s *Store) Lines(
	ctx context.Context,
	season calendar.Number,
	country standings.Country,
	from standings.Cursor,
	limit int,
) ([]standings.Line, error) {
	args := []any{int64(season), int64(from.Tiles), uuid.UUID(from.Account), limit} //nolint:gosec // a cursor's tiles fit in int64.
	query := linesColumns + ` WHERE season = $1 AND main` + linesFrom
	if country != "" {
		query = linesColumns + ` WHERE season = $1 AND main AND country = $5` + linesFrom
		args = append(args, string(country))
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read the standings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var lines []standings.Line
	for rows.Next() {
		var (
			account uuid.UUID
			country string
			tiles   int64
		)
		if err := rows.Scan(&account, &country, &tiles); err != nil {
			return nil, fmt.Errorf("failed to read a line: %w", err)
		}
		lines = append(lines, standings.Line{
			Account: standings.AccountID(account),
			Country: standings.Country(country),
			Tiles:   uint64(tiles), //nolint:gosec // CHECK (tiles > 0).
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the standings: %w", err)
	}
	return lines, nil
}

func (s *Store) DeleteAccount(ctx context.Context, account standings.AccountID) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM contributions WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to delete the account's contributions: %w", err)
	}
	return nil
}
