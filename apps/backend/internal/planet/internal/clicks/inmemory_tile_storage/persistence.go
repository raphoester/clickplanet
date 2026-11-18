package inmemory_tile_storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// ErrOtherBoard is a map whose rows name a board other than the one its era plays: ids of another board would land on
// other ground, so the boot refuses it.
var ErrOtherBoard = errors.New("the map's rows name a board other than its era's")

// Persistence keeps each map's tiles, the fortify lock of each province and the flags out: the world's, and each arena's.
// Each row names the board its tile is on.
type Persistence interface {
	// Boards are the boards a map's tiles, locks, mines and viruses name.
	Boards(ctx context.Context, played clicks.MapID) ([]clicks.BoardID, error)
	Load(ctx context.Context, played clicks.MapID, visit func(tile Tile)) error
	// LoadProvinces reads the locks of the provinces of that provinces file alone: another cut numbers them again.
	LoadProvinces(ctx context.Context, played clicks.MapID, asset string,
		visit func(province clicks.ProvinceID, fortifiedBy string)) error
	LoadOutFlags(ctx context.Context, played clicks.MapID, visit func(out OutFlag)) error
	LoadViruses(ctx context.Context, played clicks.MapID, visit func(virus clicks.VirusSnapshot)) error
	LoadMines(ctx context.Context, played clicks.MapID, visit func(mine clicks.MineSnapshot)) error
	Save(ctx context.Context, played clicks.Stage, tiles []Tile, provinces []Province, out []OutFlag) error
	// SaveViruses keeps a map's live viruses in place of the ones it kept before.
	SaveViruses(ctx context.Context, played clicks.Stage, live []clicks.VirusSnapshot) error
	// SaveMines keeps each mine laid, in place of what its tile kept, and deletes the mine of each tile gone.
	SaveMines(ctx context.Context, played clicks.Stage, laid []clicks.MineSnapshot, gone []uint32) error
	// Open writes a map's first tiles in one transaction with keep, once keep says to.
	Open(ctx context.Context, played clicks.Stage, tiles []Tile, keep func(ctx context.Context) (bool, error)) (bool, error)
	// Carry moves a map onto another board in one transaction with keep, once keep says to: every tile, province's lock,
	// virus and mine of the board before deleted, and the tiles and mines carried written on the new one. The flags out
	// stay: they name no tile.
	Carry(ctx context.Context, onto clicks.Stage, tiles []Tile, mines []clicks.MineSnapshot,
		keep func(ctx context.Context) (bool, error)) (bool, error)
	// Wipe deletes every tile, every province's lock, every flag out, every virus and every mine of a map, in one
	// transaction with keep, once keep says to.
	Wipe(ctx context.Context, played clicks.MapID, keep func(ctx context.Context) (bool, error)) (bool, error)
	// ForgetHolder takes an account off every tile of every map, and takes its mines off them.
	ForgetHolder(ctx context.Context, account cpsession.AccountID) error
}

// Placements ends each placement whose shield a flush saved, in the transaction that saved it.
type Placements interface {
	Confirm(ctx context.Context, placed []clicks.PlacementID) error
}

type Tile struct {
	ID      uint32
	Owner   string
	Shields int
	// NoAccount when no player's take holds it: an operator's write, or a take older than the holders.
	Holder cpsession.AccountID
}

// Asset is the provinces file the id indexes: a new cut numbers its provinces again.
type Province struct {
	ID          clicks.ProvinceID
	Asset       string
	FortifiedBy string
}

const flushTimeout = 10 * time.Second

// Load reads the map that plays, the world's or the open arena's, on the board its era plays: rows of another board
// refuse it.
func (s *Storage) Load(ctx context.Context, playing clicks.Stage) error {
	ground, err := s.grounds.GroundOf(playing.Board)
	if err != nil {
		return fmt.Errorf("failed to read the ground of map %s: %w", playing.Map, err)
	}

	s.staging.Lock()
	defer s.staging.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.stage = playing
	s.groundLocked(ground)
	return s.fillLocked(ctx)
}

// fillLocked puts the tiles, the locks and the flags out the map in play keeps on the board, as on a blank one, and
// settles every province. A row of the map on a board other than the stage's refuses it.
func (s *Storage) fillLocked(ctx context.Context) error {
	boards, err := s.persistence.Boards(ctx, s.stage.Map)
	if err != nil {
		return fmt.Errorf("failed to read the boards of the stored map: %w", err)
	}
	for _, board := range boards {
		if board != s.stage.Board {
			return fmt.Errorf("%w: map %s plays board %d, and its rows name board %d", ErrOtherBoard, s.stage.Map,
				s.stage.Board, board)
		}
	}
	return s.fillFromLocked(ctx, s.persistence)
}

// Rows are what a map is filled from: what postgres keeps, or what a switch of board carried.
type Rows interface {
	Load(ctx context.Context, played clicks.MapID, visit func(tile Tile)) error
	LoadProvinces(ctx context.Context, played clicks.MapID, asset string,
		visit func(province clicks.ProvinceID, fortifiedBy string)) error
	LoadOutFlags(ctx context.Context, played clicks.MapID, visit func(out OutFlag)) error
	LoadViruses(ctx context.Context, played clicks.MapID, visit func(virus clicks.VirusSnapshot)) error
	LoadMines(ctx context.Context, played clicks.MapID, visit func(mine clicks.MineSnapshot)) error
}

func (s *Storage) fillFromLocked(ctx context.Context, rows Rows) error {
	var (
		owned, outside int
		internErr      error
	)
	err := rows.Load(ctx, s.stage.Map, func(tile Tile) {
		if tile.ID > s.board.last() {
			outside++
			return
		}
		id, err := s.board.intern(tile.Owner)
		if err != nil {
			internErr = err
			return
		}
		s.board.restore(tile.ID, tileState{
			owner:   id,
			shields: uint8(min(max(tile.Shields, 0), math.MaxUint8)), //nolint:gosec // clamped to a byte.
			holder:  s.board.holders.intern(tile.Holder),
		})
		s.landmasses.moved(tile.ID, unownedCode, id)
		s.provinces.moved(tile.ID, unownedCode, id)
		owned++
	})
	if err != nil {
		return fmt.Errorf("failed to read stored tiles: %w", err)
	}
	if internErr != nil {
		return fmt.Errorf("failed to intern a stored country: %w", internErr)
	}

	fortified, strays := 0, 0
	err = rows.LoadProvinces(ctx, s.stage.Map, s.provinces.asset(), func(province clicks.ProvinceID, fortifiedBy string) {
		if !s.provinces.known(province) {
			strays++
			return
		}
		id, err := s.board.intern(fortifiedBy)
		if err != nil {
			internErr = err
			return
		}
		s.provinces.restore(province, id)
		fortified++
	})
	if err != nil {
		return fmt.Errorf("failed to read the stored provinces' locks: %w", err)
	}
	if internErr != nil {
		return fmt.Errorf("failed to intern a stored country: %w", internErr)
	}

	err = rows.LoadOutFlags(ctx, s.stage.Map, s.out.restore)
	if err != nil {
		return fmt.Errorf("failed to read the stored flags out: %w", err)
	}

	err = rows.LoadViruses(ctx, s.stage.Map, s.viruses.restore)
	if err != nil {
		return fmt.Errorf("failed to read the stored viruses: %w", err)
	}

	err = rows.LoadMines(ctx, s.stage.Map, s.mines.restore)
	if err != nil {
		return fmt.Errorf("failed to read the stored mines: %w", err)
	}

	s.provinces.each(s.settleProvinceLocked)

	if outside > 0 {
		s.logger.Warn("stored tiles past the end of the map were ignored", slog.Int("tiles", outside))
	}
	if strays > 0 {
		s.logger.Warn("stored locks of provinces past the end of the provinces were ignored", slog.Int("provinces", strays))
	}

	s.logger.Info("loaded the tile map", slog.String("map", string(s.stage.Map)), slog.Int("board", int(s.stage.Board)), slog.Int("ownedTiles", owned), slog.Int("countryCodes", s.board.countries()),
		slog.Int("fortifiedProvinces", fortified), slog.Int("flagsOut", len(s.out.flags())),
		slog.Int("viruses", len(s.viruses.live)), slog.Int("mines", len(s.mines.laid)))

	return nil
}

func (s *Storage) Name() string { return "tiles-storage" }

func (s *Storage) Run(ctx context.Context) {
	var ticking sync.WaitGroup
	ticking.Go(func() { s.publishTallyEvery(ctx, tallyInterval) })
	defer ticking.Wait()

	ticker := time.NewTicker(s.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.flushOrLog(ctx)
		case <-ctx.Done():
			s.flushOrLog(context.WithoutCancel(ctx))
			return
		}
	}
}

func (s *Storage) flushOrLog(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, flushTimeout)
	defer cancel()

	if err := s.Flush(ctx); err != nil {
		s.logger.Error("failed to flush the tile map, retrying next tick", slog.Any("error", err))
	}
}

func (s *Storage) Flush(ctx context.Context) error {
	s.flushing.Lock()
	defer s.flushing.Unlock()

	s.mu.Lock()
	playing, changes := s.stage, s.drainLocked()
	s.mu.Unlock()

	if err := s.save(ctx, playing, changes); err != nil {
		s.mu.Lock()
		s.markDirtyLocked(changes)
		s.mu.Unlock()
		return err
	}
	return nil
}

// flushLocked saves what changed while the caller holds the flush lock and the tiles lock.
func (s *Storage) flushLocked(ctx context.Context) error {
	changes := s.drainLocked()
	if err := s.save(ctx, s.stage, changes); err != nil {
		s.markDirtyLocked(changes)
		return err
	}
	return nil
}

// What changed since the last flush, saved in one transaction.
type unsaved struct {
	tiles     []Tile
	provinces []Province
	out       []OutFlag
	viruses   Viruses
	mines     Mines
	placed    []clicks.PlacementID
}

func (s *Storage) drainLocked() unsaved {
	return unsaved{
		tiles: s.board.drainDirty(), provinces: s.provinces.drainDirty(s.board.nameOf), out: s.out.drainUnsaved(),
		viruses: s.viruses.drainUnsaved(), mines: s.mines.drainUnsaved(), placed: s.drainPlacedLocked(),
	}
}

func (s *Storage) save(ctx context.Context, playing clicks.Stage, changes unsaved) error {
	if len(changes.tiles) == 0 && len(changes.provinces) == 0 && len(changes.out) == 0 && len(changes.placed) == 0 &&
		!changes.viruses.Changed && changes.mines.empty() {
		return nil
	}
	if err := s.transactioner.InTx(ctx, func(ctx context.Context) error {
		if err := s.persistence.Save(ctx, playing, changes.tiles, changes.provinces, changes.out); err != nil {
			return err //nolint:wrapcheck // wrapped once below.
		}
		if changes.viruses.Changed {
			if err := s.persistence.SaveViruses(ctx, playing, changes.viruses.Live); err != nil {
				return err //nolint:wrapcheck // wrapped once below.
			}
		}
		if !changes.mines.empty() {
			if err := s.persistence.SaveMines(ctx, playing, changes.mines.Laid, changes.mines.Gone); err != nil {
				return err //nolint:wrapcheck // wrapped once below.
			}
		}
		return s.placements.Confirm(ctx, changes.placed) //nolint:wrapcheck // wrapped once below.
	}); err != nil {
		return fmt.Errorf("failed to save %d tiles, %d provinces' locks, %d flags out and %d shields placed: %w",
			len(changes.tiles), len(changes.provinces), len(changes.out), len(changes.placed), err)
	}
	return nil
}

func (s *Storage) markDirtyLocked(changes unsaved) {
	for _, tile := range changes.tiles {
		s.board.markDirty(tile.ID)
	}
	for _, province := range changes.provinces {
		s.provinces.markDirty(province.ID)
	}
	s.out.markUnsaved(changes.out)
	s.viruses.markUnsaved(changes.viruses)
	s.mines.markUnsaved(changes.mines)
	s.placed = append(s.placed, changes.placed...)
}

func (s *Storage) drainPlacedLocked() []clicks.PlacementID {
	placed := s.placed
	s.placed = nil
	return placed
}
