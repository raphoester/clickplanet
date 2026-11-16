package postgres_tile_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func (s *Store) LoadMines(ctx context.Context, played clicks.MapID, visit func(mine clicks.MineSnapshot)) error {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT tile, flag, account, scope FROM mines WHERE map = $1 ORDER BY tile
	`, string(played))
	if err != nil {
		return fmt.Errorf("failed to read the mines: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			mine    clicks.MineSnapshot
			account uuid.UUID
		)
		if err := rows.Scan(&mine.Tile, &mine.Flag, &account, &mine.Scope); err != nil {
			return fmt.Errorf("failed to read a mine: %w", err)
		}
		mine.Account = cpsession.AccountID(account)
		visit(mine)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read the mines: %w", err)
	}

	return nil
}

func (s *Store) SaveMines(ctx context.Context, played clicks.MapID, laid []clicks.MineSnapshot, gone []uint32) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if len(gone) > 0 {
			tiles := make([]int64, len(gone))
			for i, tile := range gone {
				tiles[i] = int64(tile)
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM mines WHERE map = $1 AND tile = ANY($2)`,
				string(played), pq.Array(tiles)); err != nil {
				return fmt.Errorf("failed to delete %d mines: %w", len(gone), err)
			}
		}
		for _, mine := range laid {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO mines (map, tile, flag, account, scope) VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (map, tile) DO UPDATE SET flag = EXCLUDED.flag, account = EXCLUDED.account, scope = EXCLUDED.scope
			`, string(played), mine.Tile, mine.Flag, mine.Account.String(), mine.Scope); err != nil {
				return fmt.Errorf("failed to keep the mine on tile %d: %w", mine.Tile, err)
			}
		}
		return nil
	})
}
