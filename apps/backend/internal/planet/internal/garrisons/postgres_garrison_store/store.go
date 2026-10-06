package postgres_garrison_store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons/inmemory_garrison_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ inmemory_garrison_storage.Persistence = (*Store)(nil)

func (s *Store) Load(ctx context.Context, visit func(garrisons.Garrison)) error {
	rows, err := s.db.QueryContext(ctx, `SELECT tile, country, defenders FROM garrisons`)
	if err != nil {
		return fmt.Errorf("failed to read the garrisons: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var garrison garrisons.Garrison
		if err := rows.Scan(&garrison.Tile, &garrison.Country, &garrison.Defenders); err != nil {
			return fmt.Errorf("failed to scan a garrison: %w", err)
		}
		visit(garrison)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read the garrisons: %w", err)
	}

	return nil
}

func (s *Store) Save(ctx context.Context, changed []garrisons.Garrison) error {
	var (
		gone, tiles []int64
		countries   []string
		defenders   []int64
	)
	for _, garrison := range changed {
		if garrison.Empty() {
			gone = append(gone, int64(garrison.Tile))
			continue
		}
		tiles = append(tiles, int64(garrison.Tile))
		countries = append(countries, garrison.Country)
		defenders = append(defenders, int64(garrison.Defenders))
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to save the garrisons: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if len(gone) > 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM garrisons WHERE tile = ANY($1::integer[])`, pq.Array(gone)); err != nil {
			return fmt.Errorf("failed to delete the garrisons that fell: %w", err)
		}
	}

	if len(tiles) > 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO garrisons (tile, country, defenders)
			SELECT * FROM unnest($1::integer[], $2::text[], $3::integer[])
			ON CONFLICT (tile) DO UPDATE SET
				country = EXCLUDED.country,
				defenders = EXCLUDED.defenders
		`, pq.Array(tiles), pq.Array(countries), pq.Array(defenders)); err != nil {
			return fmt.Errorf("failed to write the garrisons: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the garrisons: %w", err)
	}

	return nil
}
