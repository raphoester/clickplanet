package inmemory_tile_storage_test

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// The old board's 4 tiles and the new board's 10: old tile 1 has children 1 to 4, 2 has 5 and 6, 3 has 8 and 9, and 4
// (a coast the new board draws as sea) none. New tiles 7 and 10 come from nowhere.
var (
	switchParents = []uint32{1, 1, 1, 1, 2, 2, 0, 3, 3, 0}
	newBoard      = clicks.TestBoard + 1
)

// A ground whose tile k is at points[k-1] on the x axis's circle, with the landmasses given.
func groundAt(name clicks.BoardName, landmasses [][]uint32, angles ...float64) *clicks.Ground {
	positions := make([]float32, 0, len(angles)*3)
	for _, angle := range angles {
		positions = append(positions, 1-float32(angle*angle)/2, float32(angle), 0)
	}
	geography, err := clicks.NewGeography(positions, nil)
	if err != nil {
		panic(err)
	}
	tiles := uint32(len(angles)) //nolint:gosec // a test's few tiles.
	bridges, err := clicks.NewBridges(tiles, nil)
	if err != nil {
		panic(err)
	}
	ground, err := clicks.NewGround(name, geography, clicks.BordersOf(tiles, landmasses...), bridges, clicks.NewCapitals(nil))
	if err != nil {
		panic(err)
	}
	return ground
}

type switchWorld struct {
	storage     *inmemory_tile_storage.Storage
	persistence *inmemory_tile_storage.MemoryPersistence
	played      *clicks.Played
	carry       clicks.Carryover
	ana         cpsession.AccountID
}

// France holds old tiles 1 and 4 (one landmass, fortified), 10 shields on 1 and 2 on 4, ana's take on 1 and a mine on
// it; Germany holds 2; a virus lives. The mine's tile is nearest new tile 3.
func (s *testSuite) switchWorld() switchWorld {
	ctx := context.Background()
	ana := cpsession.AccountID(uuid.New())
	old := groundAt("earth-1", [][]uint32{{1, 4}}, 0.10, 0.20, 0.30, 0.40)
	fresh := groundAt("earth-2", [][]uint32{{1, 2, 3, 4}},
		0.06, 0.08, 0.101, 0.13, 0.19, 0.21, 0.25, 0.29, 0.31, 0.45)
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{1: "fr", 2: "de", 4: "fr"})
	persistence.Shield(1, 10)
	persistence.Shield(4, 2)
	persistence.Hold(1, ana)
	persistence.Fortified("test-borders", 1, "fr")
	s.Require().NoError(persistence.SaveMines(ctx, clicks.WorldStage,
		[]clicks.MineSnapshot{{Tile: 1, Flag: "fr", Account: ana, Scope: "1.2.3.4"}}, nil))
	s.Require().NoError(persistence.SaveViruses(ctx, clicks.WorldStage, []clicks.VirusSnapshot{{
		ID: clicks.VirusID(uuid.NewString()), Account: ana, Flag: "fr", Victim: "de", Tile: 2, Infected: []uint32{2},
		Infections: 1, Next: time.Now().Add(time.Minute),
	}}))
	played := clicks.NewPlayed()
	storage := inmemory_tile_storage.New(clicks.GroundsOf{clicks.TestBoard: old, newBoard: fresh}, played,
		clicks.NoCapitals(), inmemory_tile_storage.Config{}, persistence, cpatomic.FakeTransactioner{},
		&inmemory_tile_storage.RecordedPlacements{}, cptime.SystemClock{}, slog.New(slog.DiscardHandler))
	s.Require().NoError(storage.Load(ctx, clicks.WorldStage))
	carry, err := clicks.NewCarryover("earth-1", "earth-2", 4, switchParents)
	s.Require().NoError(err)
	return switchWorld{storage: storage, persistence: persistence, played: played, carry: carry, ana: ana}
}

func carrying(kept bool, seen *clicks.Carried) func(context.Context, clicks.Carried) (bool, error) {
	return func(_ context.Context, carried clicks.Carried) (bool, error) {
		*seen = carried
		return kept, nil
	}
}

var wantCarried = clicks.Carried{
	From: "earth-1", To: "earth-2", Tiles: 10, Carried: 8, Blank: 2, HeldBefore: 3, HeldAfter: 6, Dropped: 1,
	DroppedHeld: 1, ShieldsBefore: 12, ShieldsAfter: 10, MinesMoved: 1, VirusesEnded: 1, LocksDropped: 1,
}

func (s *testSuite) TestASwitchCarriesEachTileOverToItsChildrenOnTheNewBoardAndPlaysIt() {
	ctx := context.Background()
	world := s.switchWorld()
	listener, before, err := world.storage.Subscribe(ctx)
	s.Require().NoError(err)
	var kept, told clicks.Carried

	carried, switched, err := world.storage.Switch(ctx, clicks.World.On(newBoard), world.carry, carrying(true, &kept),
		func(carried clicks.Carried) { told = carried })

	s.Require().NoError(err)
	s.True(switched)
	s.Equal(wantCarried, carried)
	s.Equal(wantCarried, kept, "keep is told what is carried, in the switch's transaction")
	s.Equal(wantCarried, told)
	s.Equal(clicks.World.On(newBoard), world.storage.Stage())
	s.Equal(clicks.BoardName("earth-2"), world.played.Name(), "every rule reads the new board")
	s.Equal(map[uint32]string{1: "fr", 2: "fr", 3: "fr", 4: "fr", 5: "de", 6: "de"}, world.persistence.Stored())
	s.Equal(newBoard, world.persistence.BoardOn(clicks.World), "every row names the new board")
	s.Equal(map[uint32]int{1: 3, 2: 3, 3: 2, 4: 2}, world.persistence.StoredShields(), "10 shields split 3, 3, 2, 2")
	s.Equal(map[uint32]cpsession.AccountID{1: world.ana, 2: world.ana, 3: world.ana, 4: world.ana},
		world.persistence.StoredHolders(), "each child keeps its parent's holder")
	s.Equal([]clicks.MineSnapshot{{Tile: 3, Flag: "fr", Account: world.ana, Scope: "1.2.3.4"}},
		world.persistence.StoredMinesOn(clicks.World), "the mine moves to the child nearest its tile")
	s.Empty(world.persistence.StoredVirusesOn(clicks.World), "live viruses end")
	s.Empty(world.persistence.StoredLandmasses("test-borders"), "the old board's locks are dropped")
	for tile, owner := range map[uint32]string{1: "fr", 4: "fr", 5: "de", 7: "", 10: ""} {
		got, ok := world.storage.Owner(tile)
		s.True(ok)
		s.Equal(owner, got, "tile %d", tile)
	}
	s.Equal(3, world.storage.Shields(1))
	_, still := <-listener
	s.False(still, "every stream is cut")
	_, after, err := world.storage.Subscribe(ctx)
	s.Require().NoError(err)
	s.Greater(after.Epoch, before.Epoch, "a new epoch")
}

func (s *testSuite) TestASwitchSettlesTheNewBoardsLandmassesAgain() {
	ctx := context.Background()
	world := s.switchWorld()
	var kept clicks.Carried

	_, _, err := world.storage.Switch(ctx, clicks.World.On(newBoard), world.carry, carrying(true, &kept), func(clicks.Carried) {})
	s.Require().NoError(err)

	s.Equal(map[clicks.LandmassID]string{1: "fr"}, world.storage.Fortresses(),
		"France holds the whole of the new board's landmass")
}

func (s *testSuite) TestACarriedCountAnswersWhatASwitchWouldCarryAndChangesNothing() {
	world := s.switchWorld()

	carried, err := world.storage.Carried(clicks.World.On(newBoard), world.carry)

	s.Require().NoError(err)
	s.Equal(wantCarried, carried)
	s.Equal(clicks.WorldStage, world.storage.Stage())
	s.Equal(map[uint32]string{1: "fr", 2: "de", 4: "fr"}, world.persistence.Stored())
	s.Equal(clicks.TestBoard, world.persistence.BoardOn(clicks.World))
	s.Len(world.persistence.StoredVirusesOn(clicks.World), 1)
}

func (s *testSuite) TestASwitchKeepRefusesLeavesTheMapOnItsBoard() {
	ctx := context.Background()
	world := s.switchWorld()
	_, before, err := world.storage.Subscribe(ctx)
	s.Require().NoError(err)
	var kept clicks.Carried
	recorded := false

	_, switched, err := world.storage.Switch(ctx, clicks.World.On(newBoard), world.carry, carrying(false, &kept),
		func(clicks.Carried) { recorded = true })

	s.Require().NoError(err)
	s.False(switched)
	s.False(recorded)
	s.Equal(clicks.WorldStage, world.storage.Stage())
	s.Equal(clicks.BoardName("earth-1"), world.played.Name())
	s.Equal(map[uint32]string{1: "fr", 2: "de", 4: "fr"}, world.persistence.Stored())
	_, still, err := world.storage.Subscribe(ctx)
	s.Require().NoError(err)
	s.Equal(before, still, "no new epoch")
}

func (s *testSuite) TestASwitchIsRefusedOntoTheBoardItPlaysAndWhileAnArenaPlays() {
	ctx := context.Background()
	world := s.switchWorld()
	var kept clicks.Carried

	_, _, err := world.storage.Switch(ctx, clicks.WorldStage, world.carry, carrying(true, &kept), func(clicks.Carried) {})
	s.Require().ErrorIs(err, clicks.ErrPlaysAlready)
	_, err = world.storage.Carried(clicks.WorldStage, world.carry)
	s.Require().ErrorIs(err, clicks.ErrPlaysAlready)

	_, err = world.storage.Blank(ctx, battle.On(clicks.TestBoard), noStart, opens(true))
	s.Require().NoError(err)
	_, _, err = world.storage.Switch(ctx, clicks.World.On(newBoard), world.carry, carrying(true, &kept), func(clicks.Carried) {})
	s.Require().ErrorIs(err, clicks.ErrNotInPlay)
	_, err = world.storage.Carried(clicks.World.On(newBoard), world.carry)
	s.Require().ErrorIs(err, clicks.ErrNotInPlay)
	s.Equal(battle.On(clicks.TestBoard), world.storage.Stage())
}

func (s *testSuite) TestASecondSwitchIsRefusedOnceTheMapPlaysTheNewBoard() {
	ctx := context.Background()
	world := s.switchWorld()
	var kept clicks.Carried
	_, _, err := world.storage.Switch(ctx, clicks.World.On(newBoard), world.carry, carrying(true, &kept), func(clicks.Carried) {})
	s.Require().NoError(err)

	_, switched, err := world.storage.Switch(ctx, clicks.World.On(newBoard), world.carry, carrying(true, &kept),
		func(clicks.Carried) {})

	s.Require().ErrorIs(err, clicks.ErrPlaysAlready)
	s.False(switched)
}

func (s *testSuite) TestACarryOverFromAnotherBoardIsRefused() {
	ctx := context.Background()
	world := s.switchWorld()
	wrong, err := clicks.NewCarryover("siege-1", "earth-2", 4, switchParents)
	s.Require().NoError(err)
	var kept clicks.Carried

	_, switched, err := world.storage.Switch(ctx, clicks.World.On(newBoard), wrong, carrying(true, &kept), func(clicks.Carried) {})

	s.Require().ErrorIs(err, clicks.ErrCarryover)
	s.False(switched)
	s.Equal(clicks.WorldStage, world.storage.Stage())
}
