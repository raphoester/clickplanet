package postgres_tile_store

import (
	"context"
	"fmt"
	"time"

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

// Boards are the boards a map's rows name: the boot refuses any but the one its era plays.
func (s *Store) Boards(ctx context.Context, played clicks.MapID) ([]clicks.BoardID, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT board FROM tiles WHERE map = $1
		UNION SELECT board FROM provinces WHERE map = $1
		UNION SELECT board FROM mines WHERE map = $1
		UNION SELECT board FROM viruses WHERE map = $1
		ORDER BY board
	`, string(played))
	if err != nil {
		return nil, fmt.Errorf("failed to read the boards of map %s: %w", played, err)
	}
	defer func() { _ = rows.Close() }()

	var boards []clicks.BoardID
	for rows.Next() {
		var board clicks.BoardID
		if err := rows.Scan(&board); err != nil {
			return nil, fmt.Errorf("failed to read a board of map %s: %w", played, err)
		}
		boards = append(boards, board)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the boards of map %s: %w", played, err)
	}
	return boards, nil
}

func (s *Store) Load(ctx context.Context, played clicks.MapID, visit func(tile inmemory_tile_storage.Tile)) error {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT id, country, shields, holder FROM tiles WHERE map = $1`, string(played))
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

func (s *Store) LoadProvinces(
	ctx context.Context, played clicks.MapID, asset string, visit func(province clicks.ProvinceID, fortifiedBy string),
) error {
	rows, err := s.db.On(ctx).QueryContext(ctx,
		`SELECT id, fortified_by FROM provinces WHERE map = $1 AND asset = $2`, string(played), asset)
	if err != nil {
		return fmt.Errorf("failed to read the provinces' locks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			province    clicks.ProvinceID
			fortifiedBy string
		)
		if err := rows.Scan(&province, &fortifiedBy); err != nil {
			return fmt.Errorf("failed to scan a province's lock: %w", err)
		}
		visit(province, fortifiedBy)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read the provinces' locks: %w", err)
	}

	return nil
}

func (s *Store) Save(
	ctx context.Context, played clicks.Stage, tiles []inmemory_tile_storage.Tile, provinces []inmemory_tile_storage.Province,
	out []clicks.OutFlag,
) error {
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
		if err := upsertTiles(ctx, tx, played, taken, takenBy, shields, holders); err != nil {
			return err
		}

		for start := 0; start < len(freed); start += chunkSize {
			end := min(start+chunkSize, len(freed))
			if _, err := tx.ExecContext(ctx,
				`DELETE FROM tiles WHERE map = $1 AND id = ANY($2::integer[])`,
				string(played.Map), pq.Array(freed[start:end]),
			); err != nil {
				return fmt.Errorf("failed to delete tiles: %w", err)
			}
		}

		if err := saveProvinces(ctx, tx, played, provinces); err != nil {
			return err
		}
		return saveOutFlags(ctx, tx, played.Map, out)
	})
}

func saveOutFlags(ctx context.Context, tx cppg.Querier, played clicks.MapID, out []clicks.OutFlag) error {
	if len(out) == 0 {
		return nil
	}

	flags := make([]string, len(out))
	by := make([]string, len(out))
	at := make([]string, len(out))
	for i, flag := range out {
		flags[i], by[i], at[i] = flag.Flag, flag.By, flag.At.UTC().Format(time.RFC3339Nano)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO knockouts (map, flag, by_flag, knocked_out_at)
		SELECT $1, * FROM unnest($2::text[], $3::text[], $4::timestamptz[])
		ON CONFLICT (map, flag) DO NOTHING
	`, string(played), pq.Array(flags), pq.Array(by), pq.Array(at)); err != nil {
		return fmt.Errorf("failed to keep the flags out: %w", err)
	}

	return nil
}

func (s *Store) LoadOutFlags(ctx context.Context, played clicks.MapID, visit func(out clicks.OutFlag)) error {
	rows, err := s.db.On(ctx).QueryContext(ctx,
		`SELECT flag, by_flag, knocked_out_at FROM knockouts WHERE map = $1`, string(played))
	if err != nil {
		return fmt.Errorf("failed to read the flags out: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var out clicks.OutFlag
		if err := rows.Scan(&out.Flag, &out.By, &out.At); err != nil {
			return fmt.Errorf("failed to read a flag out: %w", err)
		}
		out.At = out.At.UTC()
		visit(out)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read the flags out: %w", err)
	}

	return nil
}

func upsertTiles(
	ctx context.Context, tx cppg.Querier, played clicks.Stage, taken []int64, takenBy []string, shields []int64,
	holders []string,
) error {
	for start := 0; start < len(taken); start += chunkSize {
		end := min(start+chunkSize, len(taken))
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO tiles (map, board, id, country, shields, holder)
			SELECT $1, $2, id, country, shields, NULLIF(holder, '')::uuid
			FROM unnest($3::integer[], $4::text[], $5::smallint[], $6::text[]) AS t(id, country, shields, holder)
			ON CONFLICT (map, id) DO UPDATE SET board = EXCLUDED.board, country = EXCLUDED.country,
				shields = EXCLUDED.shields, holder = EXCLUDED.holder
		`, string(played.Map), played.Board, pq.Array(taken[start:end]), pq.Array(takenBy[start:end]),
			pq.Array(shields[start:end]), pq.Array(holders[start:end])); err != nil {
			return fmt.Errorf("failed to upsert tiles: %w", err)
		}
	}

	return nil
}

// Open writes a map's first tiles in keep's transaction, once keep says to.
func (s *Store) Open(
	ctx context.Context, played clicks.Stage, tiles []inmemory_tile_storage.Tile, keep func(ctx context.Context) (bool, error),
) (bool, error) {
	taken := make([]int64, 0, len(tiles))
	takenBy := make([]string, 0, len(tiles))
	holders := make([]string, 0, len(tiles))
	shields := make([]int64, 0, len(tiles))
	for _, tile := range tiles {
		taken = append(taken, int64(tile.ID))
		takenBy = append(takenBy, tile.Owner)
		shields = append(shields, int64(tile.Shields))
		holders = append(holders, holderOf(tile.Holder))
	}

	opened := false
	err := s.db.InTx(ctx, func(ctx context.Context) error {
		kept, err := keep(ctx)
		if err != nil || !kept {
			return err
		}
		if err := upsertTiles(ctx, s.db.On(ctx), played, taken, takenBy, shields, holders); err != nil {
			return err
		}
		opened = true
		return nil
	})
	if err != nil {
		return false, err //nolint:wrapcheck // each step names itself.
	}

	return opened, nil
}

// lib/pq takes no array of a named type, and a NULL holder is no take's.
func holderOf(account cpsession.AccountID) string {
	if account == cpsession.NoAccount {
		return ""
	}
	return account.String()
}

func saveProvinces(ctx context.Context, tx cppg.Querier, played clicks.Stage, provinces []inmemory_tile_storage.Province) error {
	if len(provinces) == 0 {
		return nil
	}

	var (
		assets, fortifiedBy []string
		ids                 []int64
	)
	for _, province := range provinces {
		assets = append(assets, province.Asset)
		ids = append(ids, int64(province.ID))
		fortifiedBy = append(fortifiedBy, province.FortifiedBy)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO provinces (map, board, asset, id, fortified_by)
		SELECT $1, $2, * FROM unnest($3::text[], $4::smallint[], $5::text[])
		ON CONFLICT (map, asset, id) DO UPDATE SET board = EXCLUDED.board, fortified_by = EXCLUDED.fortified_by
	`, string(played.Map), played.Board, pq.Array(assets), pq.Array(ids), pq.Array(fortifiedBy)); err != nil {
		return fmt.Errorf("failed to upsert the provinces' locks: %w", err)
	}

	return nil
}

func (s *Store) Wipe(ctx context.Context, played clicks.MapID, keep func(ctx context.Context) (bool, error)) (bool, error) {
	wiped := false
	err := s.db.InTx(ctx, func(ctx context.Context) error {
		kept, err := keep(ctx)
		if err != nil || !kept {
			return err
		}
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `DELETE FROM tiles WHERE map = $1`, string(played)); err != nil {
			return fmt.Errorf("failed to delete the tiles: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM provinces WHERE map = $1`, string(played)); err != nil {
			return fmt.Errorf("failed to delete the provinces' locks: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM knockouts WHERE map = $1`, string(played)); err != nil {
			return fmt.Errorf("failed to delete the flags out: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM viruses WHERE map = $1`, string(played)); err != nil {
			return fmt.Errorf("failed to delete the viruses: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM mines WHERE map = $1`, string(played)); err != nil {
			return fmt.Errorf("failed to delete the mines: %w", err)
		}
		wiped = true
		return nil
	})
	if err != nil {
		return false, err //nolint:wrapcheck // each step names itself.
	}
	return wiped, nil
}

// Carry moves a map onto another board in keep's transaction, once keep says to: the rows of the board before go, and
// the tiles carried are copied onto onto's board (COPY, a few seconds for a million), with the mines carried. The flags
// out stay.
func (s *Store) Carry(
	ctx context.Context, onto clicks.Stage, tiles []inmemory_tile_storage.Tile, mines []clicks.MineSnapshot,
	keep func(ctx context.Context) (bool, error),
) (bool, error) {
	carried := false
	err := s.db.InTx(ctx, func(ctx context.Context) error {
		kept, err := keep(ctx)
		if err != nil || !kept {
			return err
		}
		tx := s.db.On(ctx)
		for _, table := range []string{"tiles", "provinces", "viruses", "mines"} {
			if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE map = $1`, string(onto.Map)); err != nil {
				return fmt.Errorf("failed to delete the map's %s: %w", table, err)
			}
		}
		if err := copyTiles(ctx, tx, onto, tiles); err != nil {
			return err
		}
		if err := s.SaveMines(ctx, onto, mines, nil); err != nil {
			return err
		}
		carried = true
		return nil
	})
	if err != nil {
		return false, err //nolint:wrapcheck // each step names itself.
	}
	return carried, nil
}

func copyTiles(ctx context.Context, tx cppg.Querier, onto clicks.Stage, tiles []inmemory_tile_storage.Tile) error {
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("tiles", "map", "board", "id", "country", "shields", "holder"))
	if err != nil {
		return fmt.Errorf("failed to start copying the tiles: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, tile := range tiles {
		var holder any
		if tile.Holder != cpsession.NoAccount {
			holder = tile.Holder.String()
		}
		if _, err := stmt.ExecContext(ctx, string(onto.Map), int64(onto.Board), int64(tile.ID), tile.Owner,
			int64(tile.Shields), holder); err != nil {
			return fmt.Errorf("failed to copy tile %d: %w", tile.ID, err)
		}
	}
	if _, err := stmt.ExecContext(ctx); err != nil {
		return fmt.Errorf("failed to copy %d tiles: %w", len(tiles), err)
	}
	return nil
}

func (s *Store) ForgetHolder(ctx context.Context, account cpsession.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `UPDATE tiles SET holder = NULL WHERE holder = $1`,
		uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to forget a holder: %w", err)
	}
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM mines WHERE account = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to take a holder's mines off: %w", err)
	}
	return nil
}
