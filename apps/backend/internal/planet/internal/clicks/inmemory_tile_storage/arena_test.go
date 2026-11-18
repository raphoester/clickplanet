package inmemory_tile_storage_test

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const battle = clicks.MapID("battle")

var noStart = []clicks.StartTile{}

// The world as a battle finds it: an island fortified by France, one tile held by ana's take, and a German tile not
// flushed yet.
func (s *testSuite) worldBeforeABattle(persistence *inmemory_tile_storage.MemoryPersistence) (*inmemory_tile_storage.Storage, cpsession.AccountID) {
	ctx := context.Background()
	ana := cpsession.AccountID(uuid.New())
	storage := s.loadedOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Click(cpctx.AddAccountToContext(ctx, ana.String()), island[0], "fr"))
	for _, tile := range island[1:] {
		s.Require().NoError(storage.Set(ctx, tile, "fr"))
	}
	_, err := storage.Fortify(ctx, island[2], "fr", 10)
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))
	s.Require().NoError(storage.Set(ctx, 12, "de"))
	return storage, ana
}

func (s *testSuite) TestABlankKeepsTheWorldWithTheTilesNotFlushedYetAndPlaysAnEmptyMap() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage, ana := s.worldBeforeABattle(persistence)

	blanked, err := storage.Blank(ctx, battle.On(clicks.TestBoard), noStart, opens(true))

	s.Require().NoError(err)
	s.True(blanked)
	s.Equal(map[uint32]string{12: "de", island[0]: "fr", island[1]: "fr", island[2]: "fr"}, persistence.Stored())
	s.Equal(map[uint32]cpsession.AccountID{island[0]: ana}, persistence.StoredHolders())
	s.Equal(map[clicks.ProvinceID]string{1: "fr"}, persistence.StoredProvinces("test-provinces"))
	for _, tile := range append([]uint32{12}, island...) {
		owner, _ := storage.Owner(tile)
		s.Empty(owner, "tile %d", tile)
	}
	s.Require().NoError(storage.Set(ctx, 13, "it"))
	s.Require().NoError(storage.Flush(ctx))
	s.Equal(map[uint32]string{13: "it"}, persistence.StoredOn(battle), "the arena's map is saved apart")
	s.Equal(map[uint32]string{12: "de", island[0]: "fr", island[1]: "fr", island[2]: "fr"}, persistence.Stored())
}

func (s *testSuite) TestABlankKeepSaysNoToLeavesTheMap() {
	ctx := context.Background()
	storage, _ := s.worldBeforeABattle(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	_, version, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	blanked, err := storage.Blank(ctx, battle.On(clicks.TestBoard), noStart, opens(false))

	s.Require().NoError(err)
	s.False(blanked)
	owner, _ := storage.Owner(12)
	s.Equal("de", owner)
	_, still, err := storage.Subscribe(ctx)
	s.Require().NoError(err)
	s.Equal(version, still)
}

func (s *testSuite) TestTheWorldIsNoArena() {
	storage := s.newStorage(inmemory_tile_storage.Config{})

	_, blankErr := storage.Blank(context.Background(), clicks.WorldStage, noStart, opens(true))
	_, putBackErr := storage.PutBack(context.Background(), clicks.World, clicks.WorldStage, keeps(true))

	s.Require().Error(blankErr)
	s.Require().Error(putBackErr)
}

func (s *testSuite) TestAPutBackDeletesTheArenasMapAndPlaysTheWorldTileForTile() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage, ana := s.worldBeforeABattle(persistence)
	_, err := storage.Blank(ctx, battle.On(clicks.TestBoard), noStart, opens(true))
	s.Require().NoError(err)
	for _, tile := range append([]uint32{12, 13}, island...) {
		s.Require().NoError(storage.Set(ctx, tile, "it"))
	}
	_, err = storage.Fortify(ctx, island[0], "it", 3)
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))
	changes, before, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	put, err := storage.PutBack(ctx, battle, clicks.WorldStage, keeps(true))

	s.Require().NoError(err)
	s.True(put)
	for tile, owner := range map[uint32]string{12: "de", 13: "", island[0]: "fr", island[1]: "fr", island[2]: "fr"} {
		held, _ := storage.Owner(tile)
		s.Equal(owner, held, "tile %d", tile)
	}
	s.Equal(1, storage.Shields(island[1]))
	s.Equal([]clicks.Holding{{Account: ana, Flag: "fr", Tiles: 1}}, storage.HoldingsOf(1))
	s.Equal(map[clicks.ProvinceID]string{1: "fr"}, storage.Fortresses())
	s.Equal(3, storage.Held("fr"))
	s.Zero(storage.Held("it"))
	s.Empty(persistence.StoredOn(battle), "the arena's map is deleted")
	s.Empty(persistence.StoredProvincesOn(battle, "test-provinces"))

	_, open := <-changes
	s.False(open, "a client opens again and reads the world")
	_, after, err := storage.Subscribe(ctx)
	s.Require().NoError(err)
	s.Greater(after.Epoch, before.Epoch)
	s.Require().NoError(storage.Set(ctx, 13, "pt"))
	s.Require().NoError(storage.Flush(ctx))
	s.Equal(map[uint32]string{12: "de", 13: "pt", island[0]: "fr", island[1]: "fr", island[2]: "fr"}, persistence.Stored())
}

func (s *testSuite) TestAPutBackOfAnArenaThatDoesNotPlayDeletesItsMapAndLeavesMemory() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage, _ := s.worldBeforeABattle(persistence)
	s.Require().NoError(persistence.Save(ctx, clicks.MapID("stale").On(clicks.TestBoard), []inmemory_tile_storage.Tile{{ID: 5, Owner: "pt"}}, nil, nil))

	put, err := storage.PutBack(ctx, "stale", clicks.WorldStage, keeps(true))

	s.Require().NoError(err)
	s.True(put)
	s.Empty(persistence.StoredOn("stale"))
	owner, _ := storage.Owner(12)
	s.Equal("de", owner)
}

func (s *testSuite) TestAPutBackThatCannotBeKeptLeavesTheMap() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage, _ := s.worldBeforeABattle(persistence)
	_, err := storage.Blank(ctx, battle.On(clicks.TestBoard), noStart, opens(true))
	s.Require().NoError(err)
	s.Require().NoError(storage.Set(ctx, 13, "it"))
	down := errors.New("postgres is down")

	put, err := storage.PutBack(ctx, battle, clicks.WorldStage, func(context.Context) (bool, error) { return false, down })

	s.Require().ErrorIs(err, down)
	s.False(put)
	owner, _ := storage.Owner(13)
	s.Equal("it", owner, "the arena's map still plays")
}

func (s *testSuite) TestAHolderForgottenWhileAnArenaPlaysHoldsNothingWhenTheWorldIsBack() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage, ana := s.worldBeforeABattle(persistence)
	_, err := storage.Blank(ctx, battle.On(clicks.TestBoard), noStart, opens(true))
	s.Require().NoError(err)

	_, err = storage.ForgetHolder(ctx, ana)
	s.Require().NoError(err)

	s.Empty(persistence.StoredHolders())
	_, err = storage.PutBack(ctx, battle, clicks.WorldStage, keeps(true))
	s.Require().NoError(err)
	s.Empty(storage.HoldingsOf(1))
}

func (s *testSuite) TestASwapOfTheMapWaitsForAWriteAndItsRecordToEnd() {
	ctx := context.Background()
	storage := s.loadedOn(inmemory_tile_storage.Config{}, inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	writing, recorded := make(chan struct{}), make(chan struct{})
	seen := make(chan clicks.Stage, 1)
	go storage.During(func(on clicks.Stage) {
		close(writing)
		<-recorded
		seen <- on
	})
	<-writing

	blanked := make(chan error, 1)
	go func() {
		_, err := storage.Blank(ctx, battle.On(clicks.TestBoard), noStart, opens(true))
		blanked <- err
	}()
	s.Never(func() bool { return len(blanked) > 0 }, 50*time.Millisecond, 5*time.Millisecond,
		"the arena opens once the write in flight recorded itself")

	close(recorded)
	s.Equal(clicks.WorldStage, <-seen, "the write and its record ran on the world's map")
	s.Require().NoError(<-blanked)
	s.Equal(battle.On(clicks.TestBoard), storage.Stage(), "the arena plays the board the world played")
	storage.During(func(on clicks.Stage) { s.Equal(battle.On(clicks.TestBoard), on) })
}

func (s *testSuite) TestAnArenaOnABoardOfItsOwnPlaysItsGroundAndKeepsTheWorldsOnItsOwn() {
	ctx := context.Background()
	fixture := clicks.TestBoard + 2
	world := clicks.NamedGround("earth-1", borders, clicks.NewCapitals(nil))
	small := clicks.NamedGround("fixture-1", clicks.BordersOf(30), clicks.NewCapitals(nil))
	played := clicks.NewPlayed()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := inmemory_tile_storage.New(clicks.GroundsOf{clicks.TestBoard: world, fixture: small}, played, clicks.NoCapitals(),
		inmemory_tile_storage.Config{}, persistence, cpatomic.FakeTransactioner{}, &inmemory_tile_storage.RecordedPlacements{},
		cptime.SystemClock{}, slog.New(slog.DiscardHandler))
	s.Require().NoError(storage.Load(ctx, clicks.WorldStage))
	s.Require().NoError(storage.Set(ctx, 12, "de"))
	s.Equal(clicks.BoardName("earth-1"), played.Name())

	_, err := storage.Blank(ctx, battle.On(fixture), noStart, opens(true))
	s.Require().NoError(err)

	s.Equal(battle.On(fixture), storage.Stage())
	s.Equal(clicks.BoardName("fixture-1"), played.Name(), "every rule reads the arena's board")
	s.Equal(uint32(30), played.Tiles())
	s.Require().Error(storage.Set(ctx, 31, "it"), "no tile past the arena's board")
	s.Require().NoError(storage.Set(ctx, 30, "it"))
	s.Require().NoError(storage.Flush(ctx))
	s.Equal(fixture, persistence.BoardOn(battle), "its rows name its board")
	kept, err := storage.Kept(ctx, clicks.WorldStage)
	s.Require().NoError(err)
	s.Len(kept.Tiles, maxIndex*2, "the world's map kept, on its own board")
	s.Equal(clicks.TestBoard, kept.Board)
	s.Equal("de", ownerIn(kept, 12))

	_, err = storage.PutBack(ctx, battle, clicks.WorldStage, keeps(true))
	s.Require().NoError(err)

	s.Equal(clicks.WorldStage, storage.Stage())
	s.Equal(clicks.BoardName("earth-1"), played.Name(), "the world's board again")
	owner, _ := storage.Owner(12)
	s.Equal("de", owner)
}

func (s *testSuite) TestABlankOnABoardThisProcessDoesNotCarryLeavesTheMap() {
	ctx := context.Background()
	played := clicks.NewPlayed()
	storage := inmemory_tile_storage.New(clicks.GroundsOf{clicks.TestBoard: clicks.GroundOf(borders)}, played,
		clicks.NoCapitals(), inmemory_tile_storage.Config{}, inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}),
		cpatomic.FakeTransactioner{}, &inmemory_tile_storage.RecordedPlacements{}, cptime.SystemClock{},
		slog.New(slog.DiscardHandler))
	s.Require().NoError(storage.Load(ctx, clicks.WorldStage))
	s.Require().NoError(storage.Set(ctx, 12, "de"))

	blanked, err := storage.Blank(ctx, battle.On(clicks.TestBoard+5), noStart, opens(true))

	s.Require().Error(err)
	s.False(blanked)
	s.Equal(clicks.WorldStage, storage.Stage())
	owner, _ := storage.Owner(12)
	s.Equal("de", owner)
}
