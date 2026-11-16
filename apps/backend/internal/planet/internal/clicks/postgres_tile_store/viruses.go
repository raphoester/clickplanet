package postgres_tile_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func (s *Store) LoadViruses(ctx context.Context, played clicks.MapID, visit func(virus clicks.VirusSnapshot)) error {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT id, account, scope, flag, victim, tile, infected, infections, next_at FROM viruses WHERE map = $1
	`, string(played))
	if err != nil {
		return fmt.Errorf("failed to read the viruses: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			virus    clicks.VirusSnapshot
			id       uuid.UUID
			account  uuid.UUID
			infected []int64
		)
		if err := rows.Scan(&id, &account, &virus.Scope, &virus.Flag, &virus.Victim, &virus.Tile, pq.Array(&infected),
			&virus.Infections, &virus.Next); err != nil {
			return fmt.Errorf("failed to read a virus: %w", err)
		}
		virus.ID, virus.Account, virus.Next = clicks.VirusID(id.String()), cpsession.AccountID(account), virus.Next.UTC()
		for _, tile := range infected {
			virus.Infected = append(virus.Infected, uint32(tile)) //nolint:gosec // a tile id, written from a uint32.
		}
		visit(virus)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read the viruses: %w", err)
	}

	return nil
}

func (s *Store) SaveViruses(ctx context.Context, played clicks.MapID, live []clicks.VirusSnapshot) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `DELETE FROM viruses WHERE map = $1`, string(played)); err != nil {
			return fmt.Errorf("failed to clear the viruses kept: %w", err)
		}
		for _, virus := range live {
			infected := make([]int64, len(virus.Infected))
			for i, tile := range virus.Infected {
				infected[i] = int64(tile)
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO viruses (map, id, account, scope, flag, victim, tile, infected, infections, next_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			`, string(played), string(virus.ID), virus.Account.String(), virus.Scope, virus.Flag, virus.Victim, virus.Tile,
				pq.Array(infected), virus.Infections, virus.Next); err != nil {
				return fmt.Errorf("failed to keep virus %s: %w", virus.ID, err)
			}
		}
		return nil
	})
}
