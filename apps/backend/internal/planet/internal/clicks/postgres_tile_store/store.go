package postgres_tile_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const chunkSize = 10_000

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ inmemory_tile_storage.Persistence = (*Store)(nil)

func (s *Store) Load(ctx context.Context, visit func(tile inmemory_tile_storage.Tile)) error {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT id, country, shields, holder FROM tiles`)
	if err != nil {
		return fmt.Errorf("failed to read tiles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			tile   inmemory_tile_storage.Tile
			holder uuid.NullUUID
		)
		if err := rows.Scan(&tile.ID, &tile.Owner, &tile.Shields, &holder); err != nil {
			return fmt.Errorf("failed to scan a tile: %w", err)
		}
		if holder.Valid {
			tile.Holder = cpsession.AccountID(holder.UUID)
		}
		visit(tile)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read tiles: %w", err)
	}

	return nil
}

func (s *Store) LoadLandmasses(
	ctx context.Context, asset string, visit func(landmass clicks.LandmassID, fortifiedBy string),
) error {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT id, fortified_by FROM landmasses WHERE asset = $1`, asset)
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
		taken, freed     []int64
		takenBy, holders []string
		shields          []int64
	)
	for _, tile := range tiles {
		if tile.Owner == "" {
			freed = append(freed, int64(tile.ID))
			continue
		}
		taken = append(taken, int64(tile.ID))
		takenBy = append(takenBy, tile.Owner)
		shields = append(shields, int64(tile.Shields))
		holders = append(holders, holderOf(tile.Holder))
	}

	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		for start := 0; start < len(taken); start += chunkSize {
			end := min(start+chunkSize, len(taken))
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO tiles (id, country, shields, holder)
				SELECT id, country, shields, NULLIF(holder, '')::uuid
				FROM unnest($1::integer[], $2::text[], $3::smallint[], $4::text[]) AS t(id, country, shields, holder)
				ON CONFLICT (id) DO UPDATE SET country = EXCLUDED.country, shields = EXCLUDED.shields, holder = EXCLUDED.holder
			`, pq.Array(taken[start:end]), pq.Array(takenBy[start:end]), pq.Array(shields[start:end]),
				pq.Array(holders[start:end])); err != nil {
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

		return saveLandmasses(ctx, tx, landmasses)
	})
}

// lib/pq takes no array of a named type, and a NULL holder is no take's.
func holderOf(account cpsession.AccountID) string {
	if account == cpsession.NoAccount {
		return ""
	}
	return account.String()
}

func saveLandmasses(ctx context.Context, tx cppg.Querier, landmasses []inmemory_tile_storage.Landmass) error {
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

func (s *Store) Wipe(ctx context.Context, keep func(ctx context.Context) (bool, error)) (bool, error) {
	wiped := false
	err := s.db.InTx(ctx, func(ctx context.Context) error {
		kept, err := keep(ctx)
		if err != nil || !kept {
			return err
		}
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `DELETE FROM tiles`); err != nil {
			return fmt.Errorf("failed to delete the tiles: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM landmasses`); err != nil {
			return fmt.Errorf("failed to delete the landmasses' locks: %w", err)
		}
		wiped = true
		return nil
	})
	if err != nil {
		return false, err //nolint:wrapcheck // each step names itself.
	}
	return wiped, nil
}
