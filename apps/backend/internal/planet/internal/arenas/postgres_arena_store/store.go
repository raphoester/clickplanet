package postgres_arena_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const chunkSize = 10_000

var _ arenas.Store = (*Store)(nil)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

func (s *Store) Find(ctx context.Context, id arenas.ID) (arenas.Arena, error) {
	arena := arenas.Arena{ID: id}
	var (
		endsAt, openedAt sql.NullTime
		closed           bool
	)
	err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT a.ends_at, a.opened_at, EXISTS (SELECT 1 FROM closed_arenas WHERE id = $1)
		FROM (SELECT 1) AS one
		LEFT JOIN arenas a ON a.id = $1
	`, string(id)).Scan(&endsAt, &openedAt, &closed)
	if err != nil {
		return arenas.Arena{}, fmt.Errorf("failed to read arena %s: %w", id, err)
	}
	if openedAt.Valid {
		arena.Opened = true
		arena.Opening = arenas.Opening{ID: id, EndsAt: endsAt.Time.UTC(), OpenedAt: openedAt.Time.UTC()}
	}
	arena.Closed = closed
	return arena, nil
}

func (s *Store) Playing(ctx context.Context) (arenas.Arena, error) {
	var (
		id               string
		endsAt, openedAt time.Time
	)
	err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT id, ends_at, opened_at FROM arenas a
		WHERE NOT EXISTS (SELECT 1 FROM closed_arenas c WHERE c.id = a.id)
		ORDER BY opened_at DESC
		LIMIT 1
	`).Scan(&id, &endsAt, &openedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return arenas.Arena{}, nil
	}
	if err != nil {
		return arenas.Arena{}, fmt.Errorf("failed to read the arena open: %w", err)
	}
	opening := arenas.Opening{ID: arenas.ID(id), EndsAt: endsAt.UTC(), OpenedAt: openedAt.UTC()}
	return arenas.Arena{ID: opening.ID, Opening: opening, Opened: true}, nil
}

func (s *Store) Open(ctx context.Context, opening arenas.Opening, before clicks.MapCopy) error {
	return s.db.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // each step names itself.
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `INSERT INTO arenas (id, ends_at, opened_at) VALUES ($1, $2, $3)`,
			string(opening.ID), opening.EndsAt, opening.OpenedAt); err != nil {
			return fmt.Errorf("failed to keep arena %s: %w", opening.ID, err)
		}
		if err := keepTiles(ctx, tx, opening.ID, before.Tiles); err != nil {
			return err
		}
		return keepLandmasses(ctx, tx, opening.ID, before.Landmasses)
	})
}

func keepTiles(ctx context.Context, tx cppg.Querier, arena arenas.ID, tiles []clicks.TileCopy) error {
	for start := 0; start < len(tiles); start += chunkSize {
		chunk := tiles[start:min(start+chunkSize, len(tiles))]
		ids, owners, shields, holders := make([]int64, len(chunk)), make([]string, len(chunk)), make([]int64, len(chunk)),
			make([]string, len(chunk))
		for i, tile := range chunk {
			ids[i], owners[i], shields[i], holders[i] = int64(tile.ID), tile.Owner, int64(tile.Shields), holderOf(tile.Holder)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO arena_tiles (arena, id, country, shields, holder)
			SELECT $1, id, country, shields, NULLIF(holder, '')::uuid
			FROM unnest($2::integer[], $3::text[], $4::smallint[], $5::text[]) AS t(id, country, shields, holder)
		`, string(arena), pq.Array(ids), pq.Array(owners), pq.Array(shields), pq.Array(holders)); err != nil {
			return fmt.Errorf("failed to keep the tiles of arena %s: %w", arena, err)
		}
	}
	return nil
}

func keepLandmasses(ctx context.Context, tx cppg.Querier, arena arenas.ID, landmasses []clicks.LandmassCopy) error {
	if len(landmasses) == 0 {
		return nil
	}
	assets, ids, fortifiedBy := make([]string, len(landmasses)), make([]int64, len(landmasses)), make([]string, len(landmasses))
	for i, landmass := range landmasses {
		assets[i], ids[i], fortifiedBy[i] = landmass.Asset, int64(landmass.ID), landmass.FortifiedBy
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO arena_landmasses (arena, asset, id, fortified_by)
		SELECT $1, * FROM unnest($2::text[], $3::smallint[], $4::text[])
	`, string(arena), pq.Array(assets), pq.Array(ids), pq.Array(fortifiedBy)); err != nil {
		return fmt.Errorf("failed to keep the landmasses of arena %s: %w", arena, err)
	}
	return nil
}

// lib/pq takes no array of a named type, and a NULL holder is no take's.
func holderOf(account cpsession.AccountID) string {
	if account == cpsession.NoAccount {
		return ""
	}
	return account.String()
}

func (s *Store) Close(ctx context.Context, closing arenas.Closing) (clicks.MapCopy, error) {
	var before clicks.MapCopy
	err := s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `INSERT INTO closed_arenas (id, closed_at) VALUES ($1, $2)`,
			string(closing.ID), closing.ClosedAt); err != nil {
			return fmt.Errorf("failed to keep arena %s closed: %w", closing.ID, err)
		}
		tiles, err := readTiles(ctx, tx, closing.ID)
		if err != nil {
			return err
		}
		landmasses, err := readLandmasses(ctx, tx, closing.ID)
		if err != nil {
			return err
		}
		before = clicks.MapCopy{Tiles: tiles, Landmasses: landmasses}
		for _, table := range []string{"arena_tiles", "arena_landmasses"} {
			if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE arena = $1`, string(closing.ID)); err != nil {
				return fmt.Errorf("failed to forget the map of arena %s: %w", closing.ID, err)
			}
		}
		return nil
	})
	if err != nil {
		return clicks.MapCopy{}, err //nolint:wrapcheck // each step names itself.
	}
	return before, nil
}

func readTiles(ctx context.Context, tx cppg.Querier, arena arenas.ID) ([]clicks.TileCopy, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, country, shields, holder FROM arena_tiles WHERE arena = $1`, string(arena))
	if err != nil {
		return nil, fmt.Errorf("failed to read the tiles of arena %s: %w", arena, err)
	}
	defer func() { _ = rows.Close() }()

	var tiles []clicks.TileCopy
	for rows.Next() {
		var (
			tile   clicks.TileCopy
			holder uuid.NullUUID
		)
		if err := rows.Scan(&tile.ID, &tile.Owner, &tile.Shields, &holder); err != nil {
			return nil, fmt.Errorf("failed to scan a tile of arena %s: %w", arena, err)
		}
		if holder.Valid {
			tile.Holder = cpsession.AccountID(holder.UUID)
		}
		tiles = append(tiles, tile)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the tiles of arena %s: %w", arena, err)
	}
	return tiles, nil
}

func readLandmasses(ctx context.Context, tx cppg.Querier, arena arenas.ID) ([]clicks.LandmassCopy, error) {
	rows, err := tx.QueryContext(ctx, `SELECT asset, id, fortified_by FROM arena_landmasses WHERE arena = $1`, string(arena))
	if err != nil {
		return nil, fmt.Errorf("failed to read the landmasses of arena %s: %w", arena, err)
	}
	defer func() { _ = rows.Close() }()

	var landmasses []clicks.LandmassCopy
	for rows.Next() {
		var landmass clicks.LandmassCopy
		if err := rows.Scan(&landmass.Asset, &landmass.ID, &landmass.FortifiedBy); err != nil {
			return nil, fmt.Errorf("failed to scan a landmass of arena %s: %w", arena, err)
		}
		landmasses = append(landmasses, landmass)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the landmasses of arena %s: %w", arena, err)
	}
	return landmasses, nil
}

func (s *Store) ForgetHolder(ctx context.Context, account cpsession.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `UPDATE arena_tiles SET holder = NULL WHERE holder = $1`,
		uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to forget a holder in the arenas: %w", err)
	}
	return nil
}
