package postgres_tile_store

import (
	"context"
	"fmt"

	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func insertSeats(ctx context.Context, tx cppg.Querier, played clicks.Stage, seats []clicks.Capital) error {
	if len(seats) == 0 {
		return nil
	}
	flags := make([]string, len(seats))
	tiles := make([]int64, len(seats))
	for i, seat := range seats {
		flags[i], tiles[i] = seat.Country, int64(seat.Tile)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO seats (map, board, flag, tile)
		SELECT $1, $2, * FROM unnest($3::text[], $4::integer[])
	`, string(played.Map), played.Board, pq.Array(flags), pq.Array(tiles)); err != nil {
		return fmt.Errorf("failed to keep the seats: %w", err)
	}
	return nil
}

func (s *Store) LoadSeats(ctx context.Context, played clicks.MapID, visit func(seat clicks.Capital)) error {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT flag, tile FROM seats WHERE map = $1 ORDER BY flag`, string(played))
	if err != nil {
		return fmt.Errorf("failed to read the seats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var seat clicks.Capital
		if err := rows.Scan(&seat.Country, &seat.Tile); err != nil {
			return fmt.Errorf("failed to read a seat: %w", err)
		}
		visit(seat)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read the seats: %w", err)
	}
	return nil
}
