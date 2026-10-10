package postgres_tile_store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const chunkSize = 10_000

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ inmemory_tile_storage.Persistence = (*Store)(nil)

func (s *Store) Load(ctx context.Context, visit func(tile uint32, owner string, shields int)) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, country, shields FROM tiles`)
	if err != nil {
		return fmt.Errorf("failed to read tiles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			tile    uint32
			owner   string
			shields int
		)
		if err := rows.Scan(&tile, &owner, &shields); err != nil {
			return fmt.Errorf("failed to scan a tile: %w", err)
		}
		visit(tile, owner, shields)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read tiles: %w", err)
	}

	return nil
}

func (s *Store) LoadLandmasses(
	ctx context.Context, asset string, visit func(landmass clicks.LandmassID, fortifiedBy string),
) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, fortified_by FROM landmasses WHERE asset = $1`, asset)
	if err != nil {
		return fmt.Errorf("failed to read landmasses: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			landmass    clicks.LandmassID
			fortifiedBy string
		)
		if err := rows.Scan(&landmass, &fortifiedBy); err != nil {
			return fmt.Errorf("failed to scan a landmass: %w", err)
		}
		visit(landmass, fortifiedBy)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read landmasses: %w", err)
	}

	return nil
}

func (s *Store) Save(ctx context.Context, tiles []inmemory_tile_storage.Tile, landmasses []inmemory_tile_storage.Landmass) error {
	var (
		taken, freed []int64
		takenBy      []string
		shields      []int64
	)
	for _, tile := range tiles {
		if tile.Owner == "" {
			freed = append(freed, int64(tile.ID))
			continue
		}
		taken = append(taken, int64(tile.ID))
		takenBy = append(takenBy, tile.Owner)
		shields = append(shields, int64(tile.Shields))
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to save tiles: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for start := 0; start < len(taken); start += chunkSize {
		end := min(start+chunkSize, len(taken))
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO tiles (id, country, shields)
			SELECT * FROM unnest($1::integer[], $2::text[], $3::smallint[])
			ON CONFLICT (id) DO UPDATE SET country = EXCLUDED.country, shields = EXCLUDED.shields
		`, pq.Array(taken[start:end]), pq.Array(takenBy[start:end]), pq.Array(shields[start:end])); err != nil {
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

	if err := saveLandmasses(ctx, tx, landmasses); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tiles: %w", err)
	}

	return nil
}

func saveLandmasses(ctx context.Context, tx *sql.Tx, landmasses []inmemory_tile_storage.Landmass) error {
	if len(landmasses) == 0 {
		return nil
	}

	var (
		assets, fortifiedBy []string
		ids                 []int64
	)
	for _, landmass := range landmasses {
		assets = append(assets, landmass.Asset)
		ids = append(ids, int64(landmass.ID))
		fortifiedBy = append(fortifiedBy, landmass.FortifiedBy)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO landmasses (asset, id, fortified_by)
		SELECT * FROM unnest($1::text[], $2::smallint[], $3::text[])
		ON CONFLICT (asset, id) DO UPDATE SET fortified_by = EXCLUDED.fortified_by
	`, pq.Array(assets), pq.Array(ids), pq.Array(fortifiedBy)); err != nil {
		return fmt.Errorf("failed to upsert landmasses: %w", err)
	}

	return nil
}
