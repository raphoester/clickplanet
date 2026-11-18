package inmemory_tile_storage

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// New is a tile map that plays the ground of its stage's board, and makes it the one played: before Load, the world on
// no board, which a test's one ground answers for, and production's grounds do not (an empty map until Load).
func New(
	grounds clicks.Grounds,
	played *clicks.Played,
	capitals clicks.CapitalPlay,
	config Config,
	persistence Persistence,
	transactioner cpatomic.Transactioner,
	placements Placements,
	clock cptime.Clock,
	logger *slog.Logger,
) *Storage {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	config = config.withDefaults()

	storage := &Storage{
		clock:         clock,
		config:        config,
		logger:        logger,
		persistence:   persistence,
		grounds:       grounds,
		played:        played,
		capitals:      capitals,
		transactioner: transactioner,
		placements:    placements,
		stage:         clicks.Stage{Map: clicks.World},
		fuse:          make(chan struct{}, 1),
		version:       clicks.MapVersion{Epoch: uint32(clock.Now().Unix())}, //nolint:gosec // seconds fit a uint32 until 2106.
		feed:          newFeed(config.SubscriberBuffer, logger),
	}
	ground, err := grounds.GroundOf(storage.stage.Board)
	if err != nil {
		ground = nil
	}
	storage.groundLocked(ground)
	return storage
}

type Storage struct {
	clock         cptime.Clock
	config        Config
	logger        *slog.Logger
	persistence   Persistence
	grounds       clicks.Grounds
	capitals      clicks.CapitalPlay
	transactioner cpatomic.Transactioner
	placements    Placements

	// Under mu: the shields placed since the last flush, confirmed by the flush that saves their tiles.
	placed []clicks.PlacementID

	// Held by a swap of the stage, and read-held by a write and its record (During), so no swap comes between them.
	// Taken before flushing and mu.
	staging sync.RWMutex

	// Held through a flush, from what it takes to what it saves, so a wipe never meets a save of the map before it.
	flushing sync.Mutex

	// The ground of the stage's board, swapped with the stage: what every rule reads the board in play by.
	played *clicks.Played

	// Guards the stage in play, the board, the landmasses and the version, which change together, and the feed's order.
	// The stage changes under staging too.
	mu         sync.RWMutex
	stage      clicks.Stage
	board      *board
	landmasses *landmasses
	out        *outFlags
	viruses    *virusBook
	mines      *mineBook
	recount    recount
	version    clicks.MapVersion

	feed *feed

	// A value once a take set a mine off, until the explosion runs.
	fuse chan struct{}
}

var _ clicks.TileStorage = (*Storage)(nil)

// Stage is the map that plays, on its board.
func (s *Storage) Stage() clicks.Stage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stage
}

// During runs write on the stage that plays, and hands it that stage: no swap of the map or of its board comes between
// a write write makes and what it records of it. write never calls During again, and holds no lock of the storage.
func (s *Storage) During(write func(on clicks.Stage)) {
	s.staging.RLock()
	defer s.staging.RUnlock()

	write(s.stage)
}

func (s *Storage) tileLocked(id uint32) (clicks.Tile, error) {
	if id > s.board.last() {
		return clicks.Tile{}, fmt.Errorf("%w: %d (max %d)", clicks.ErrTileOutOfRange, id, s.board.last())
	}

	return s.board.tile(id).Restore(), nil
}

func (s *Storage) writeLocked(tile clicks.Tile, clicked bool) (clicks.TileUpdate, bool, error) {
	written := tile.TakeSnapshot()
	if written.ID > s.board.last() {
		return clicks.TileUpdate{}, false, fmt.Errorf("%w: %d (max %d)", clicks.ErrTileOutOfRange, written.ID, s.board.last())
	}

	owner, err := s.board.intern(written.Owner)
	if err != nil {
		return clicks.TileUpdate{}, false, err
	}
	shields := uint8(written.Shields) //nolint:gosec // a claim only takes shields away.
	state := tileState{owner: owner, shields: shields, holder: s.board.holders.intern(written.Holder)}
	before := s.board.tiles[written.ID]
	if state == before {
		return clicks.TileUpdate{}, false, nil
	}
	s.putLocked(written.ID, state)

	return clicks.TileUpdate{
		Tile: written.ID, Value: written.Owner, Previous: s.board.nameOf(before.owner), Clicked: clicked && owner != before.owner,
		Shields: written.Shields,
	}, true, nil
}

// An operator's write or a blast: the shields go home and no player's take holds the tile.
func (s *Storage) moveLocked(tile uint32, to uint16) {
	s.putLocked(tile, tileState{owner: to})
}

func (s *Storage) putLocked(tile uint32, state tileState) {
	from := s.board.put(tile, state)
	if from != state.owner {
		s.viruses.cure(tile)
		s.mines.take(tile)
		s.landmasses.moved(tile, from, state.owner)
		s.recount.moved(s.landmasses.of(tile), from, state.owner)
	}
}

// Under s.mu: every stream then gets the changes in the order of their versions.
func (s *Storage) publishLocked(change clicks.Change) {
	s.version.Seq++
	change.Version = s.version
	s.feed.publish(change)
}

func (s *Storage) Clear(_ context.Context, blast clicks.Blast) (clicks.Blast, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, tile := range blast.Cleared {
		if tile > s.board.last() {
			return clicks.Blast{}, fmt.Errorf("tile %d out of range (max %d)", tile, s.board.last())
		}
	}

	cleared := make([]uint32, 0, len(blast.Cleared))
	owners := make([]string, 0, len(blast.Cleared))
	var struck []uint32
	var left []int
	var struckOwners []string

	for _, tile := range blast.Cleared {
		s.mines.take(tile)
		if s.board.shieldsOn(tile) > 0 {
			struck = append(struck, tile)
			struckOwners = append(struckOwners, s.board.ownerOf(tile))
			left = append(left, s.board.strike(tile))
			continue
		}
		if owner := s.board.ownerOf(tile); owner != "" {
			owners = append(owners, owner)
			s.moveLocked(tile, unownedCode)
			cleared = append(cleared, tile)
		}
	}

	blast.Cleared = cleared
	blast.Owners = owners
	blast.Struck = struck
	blast.Left = left
	blast.StruckOwners = struckOwners
	s.publishLocked(clicks.Change{Blast: &blast})

	return blast, nil
}

func (s *Storage) Share(country string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return float64(s.board.heldBy(country)) / float64(s.board.last())
}

func (s *Storage) Held(country string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return int(s.board.heldBy(country))
}

func (s *Storage) Owner(tile uint32) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if tile > s.board.last() {
		return "", false
	}

	return s.board.ownerOf(tile), true
}

func (s *Storage) Subscribe(ctx context.Context) (<-chan clicks.Change, clicks.MapVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.feed.subscribe(ctx), s.version, nil
}

func (s *Storage) Shields(tile uint32) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if tile > s.board.last() {
		return 0
	}

	return s.board.shieldsOn(tile)
}

func (s *Storage) Shield(_ context.Context, tile uint32, country string, most int, placement clicks.PlacementID) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tile > s.board.last() {
		return 0, fmt.Errorf("%w: %d", clicks.ErrTileOutOfRange, tile)
	}
	if s.capitalLocked(tile) {
		return 0, fmt.Errorf("tile %d: %w", tile, clicks.ErrCapital)
	}
	if err := clicks.ShieldError(s.board.ownerOf(tile), country, s.board.shieldsOn(tile), min(most, math.MaxUint8)); err != nil {
		return 0, fmt.Errorf("tile %d: %w", tile, err)
	}
	now := s.board.raise(tile)
	s.placed = append(s.placed, placement)
	s.publishLocked(clicks.Change{Update: &clicks.TileUpdate{Tile: tile, Value: country, Previous: country, Shields: now}})

	return now, nil
}

func (s *Storage) StateBatchDense(start uint32, end uint32) (clicks.DenseBatch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	end = min(end, s.board.last())
	if start > end {
		return clicks.DenseBatch{}, fmt.Errorf("invalid tile range %d..%d", start, end)
	}

	batch := s.board.batch(start, end)
	batch.Version = s.version
	batch.Board = s.stage.Board
	batch.KnockedOut = s.out.flags()
	batch.Infected = s.viruses.infectedIn(start, end)

	return batch, nil
}

// HoldsAny says whether the country holds one of the tiles.
func (s *Storage) HoldsAny(country string, tiles []uint32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	code, ok := s.board.codeOf(country)
	if !ok {
		return false
	}
	for _, tile := range tiles {
		if tile <= s.board.last() && s.board.tiles[tile].owner == code {
			return true
		}
	}

	return false
}

// A capital whose shields the rules fix keeps the shields it has: no placement and no fortify adds one, whoever holds
// it.
func (s *Storage) capitalLocked(tile uint32) bool {
	return s.capitals.Fixes(tile)
}
