// Package postgres_tile_store keeps the in-memory tile map between boots: one row per owned tile.
package postgres_tile_store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// A reassign can move a whole country at once; chunks keep one statement's arrays a sane size.
const chunkSize = 10_000

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

// Load calls visit once per owned tile, in no particular order.
func (s *Store) Load(ctx context.Context, visit func(tile uint32, owner string)) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, country FROM tiles`)
	if err != nil {
		return fmt.Errorf("failed to read tiles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			tile  uint32
			owner string
		)
		if err := rows.Scan(&tile, &owner); err != nil {
			return fmt.Errorf("failed to scan a tile: %w", err)
		}
		visit(tile, owner)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read tiles: %w", err)
	}

	return nil
}

// Save writes owners[i] as the owner of tiles[i] in one transaction; an empty owner deletes the row.
func (s *Store) Save(ctx context.Context, tiles []uint32, owners []string) error {
	if len(tiles) != len(owners) {
		return fmt.Errorf("saving %d tiles with %d owners", len(tiles), len(owners))
	}

	var (
		taken, freed []int64
		takenBy      []string
	)
	for i, tile := range tiles {
		if owners[i] == "" {
			freed = append(freed, int64(tile))
			continue
		}
		taken = append(taken, int64(tile))
		takenBy = append(takenBy, owners[i])
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to save tiles: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for start := 0; start < len(taken); start += chunkSize {
		end := min(start+chunkSize, len(taken))
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO tiles (id, country)
			SELECT * FROM unnest($1::integer[], $2::text[])
			ON CONFLICT (id) DO UPDATE SET country = EXCLUDED.country
		`, pq.Array(taken[start:end]), pq.Array(takenBy[start:end])); err != nil {
			return fmt.Errorf("failed to upsert tiles: %w", err)
		}
	}

	for start := 0; start < len(freed); start += chunkSize {
		end := min(start+chunkSize, len(freed))
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM tiles WHERE id = ANY($1::integer[])`,
			pq.Array(freed[start:end]),
		); err != nil {
			return fmt.Errorf("failed to delete tiles: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tiles: %w", err)
	}

	return nil
}
