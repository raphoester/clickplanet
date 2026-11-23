package postgres_tile_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func (s *Store) LoadTraps(ctx context.Context, played clicks.MapID, visit func(trap clicks.TrapSnapshot)) error {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT tile, flag, account FROM traps WHERE map = $1 ORDER BY tile
	`, string(played))
	if err != nil {
		return fmt.Errorf("failed to read the traps: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			trap    clicks.TrapSnapshot
			account uuid.UUID
		)
		if err := rows.Scan(&trap.Tile, &trap.Flag, &account); err != nil {
			return fmt.Errorf("failed to read a trap: %w", err)
		}
		trap.Account = cpsession.AccountID(account)
		visit(trap)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read the traps: %w", err)
	}

	return nil
}

func (s *Store) SaveTraps(ctx context.Context, played clicks.Stage, laid []clicks.TrapSnapshot, gone []uint32) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if len(gone) > 0 {
			tiles := make([]int64, len(gone))
			for i, tile := range gone {
				tiles[i] = int64(tile)
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM traps WHERE map = $1 AND tile = ANY($2)`,
				string(played.Map), pq.Array(tiles)); err != nil {
				return fmt.Errorf("failed to delete %d traps: %w", len(gone), err)
			}
		}
		for _, trap := range laid {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO traps (map, board, tile, flag, account) VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (map, tile) DO UPDATE SET board = EXCLUDED.board, flag = EXCLUDED.flag,
					account = EXCLUDED.account
			`, string(played.Map), played.Board, trap.Tile, trap.Flag, trap.Account.String()); err != nil {
				return fmt.Errorf("failed to keep the trap on tile %d: %w", trap.Tile, err)
			}
		}
		return nil
	})
}
