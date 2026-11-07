package postgres_contribution_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ standings.Store = (*Store)(nil)

const takeLock = 0x73656173

// RecordTakes writes only the rows whose tiles changed, each with the tally's new mark: the tally's mark is its highest
// row's.
func (s *Store) RecordTakes(ctx context.Context, book books.BookID, account standings.AccountID, takes []standings.Take) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, takeLock, account.String()); err != nil {
			return fmt.Errorf("failed to lock the account's contributions: %w", err)
		}

		tally, err := tallyOf(ctx, tx, book, account)
		if err != nil {
			return err
		}
		next := tally.WithTakes(takes)

		var (
			countries []string
			tiles     []int64
			main      []bool
		)
		for country, held := range next.Tiles {
			if held != tally.Tiles[country] {
				countries = append(countries, string(country))
				tiles = append(tiles, int64(held)) //nolint:gosec // one a tile taken: never past int64.
				main = append(main, country == next.Main)
			}
		}
		if len(countries) == 0 {
			return nil
		}

		// The old main row goes first: a row made main while another still is breaks contributions_main_key.
		if tally.Main != "" && next.Main != tally.Main {
			if _, err := tx.ExecContext(ctx, `
				UPDATE contributions SET main = false WHERE book_id = $1 AND account_id = $2 AND country = $3
			`, string(book), uuid.UUID(account), string(tally.Main)); err != nil {
				return fmt.Errorf("failed to move the main flag: %w", err)
			}
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO contributions (book_id, account_id, country, tiles, main, counted_takes_below, counted_takes_below_index)
			SELECT $1, $2, taken.country, taken.tiles, taken.main, $6, $7
			FROM unnest($3::text[], $4::bigint[], $5::boolean[]) AS taken (country, tiles, main)
			ON CONFLICT (book_id, account_id, country) DO UPDATE
			SET tiles = excluded.tiles, main = excluded.main, counted_takes_below = excluded.counted_takes_below,
				counted_takes_below_index = excluded.counted_takes_below_index
		`, string(book), uuid.UUID(account), pq.Array(countries), pq.Array(tiles), pq.Array(main),
			int64(next.TakesBelow.Position), int64(next.TakesBelow.Index)); err != nil { //nolint:gosec // a ledger position fits in int64.
			return fmt.Errorf("failed to save the takes: %w", err)
		}
		return nil
	})
}

func tallyOf(ctx context.Context, db cppg.Querier, book books.BookID, account standings.AccountID) (standings.Tally, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT country, tiles, main, counted_takes_below, counted_takes_below_index FROM contributions
		WHERE book_id = $1 AND account_id = $2 ORDER BY counted_takes_below, counted_takes_below_index
	`, string(book), uuid.UUID(account))
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
			below   int64
			index   int64
		)
		if err := rows.Scan(&country, &tiles, &main, &below, &index); err != nil {
			return standings.Tally{}, fmt.Errorf("failed to read a contribution: %w", err)
		}
		tally.Tiles[standings.Country(country)] = uint64(tiles)                              //nolint:gosec // CHECK (tiles > 0).
		tally.TakesBelow = standings.TakeMark{Position: uint64(below), Index: uint32(index)} //nolint:gosec // CHECK (>= 0); the last row's mark is the highest.
		if main {
			tally.Main = standings.Country(country)
		}
	}
	if err := rows.Err(); err != nil {
		return standings.Tally{}, fmt.Errorf("failed to read the account's contributions: %w", err)
	}
	return tally, nil
}

func (s *Store) DeleteAccount(ctx context.Context, account standings.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM contributions WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to delete the account's contributions: %w", err)
	}
	return nil
}
