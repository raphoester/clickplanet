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

func (s *testSuite) TestLoadRestoresTheMapAndTheShares() {
	rows := map[uint32]string{1: "bg", 2: "bg", 3: "fr", maxIndex: "de"}
	persistence := inmemory_tile_storage.NewMemoryPersistence(rows)

	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Load(context.Background(), clicks.WorldStage))

	state, err := stateBatch(storage, 0, maxIndex)
	s.Require().NoError(err)
	s.Equal(rows, state)
	s.InDelta(2.0/maxIndex, storage.Share("bg"), 1e-12)
	s.InDelta(1.0/maxIndex, storage.Share("fr"), 1e-12)
}

func (s *testSuite) TestLoadSkipsTilesPastTheEndOfTheMap() {
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{10: "fr", maxIndex + 1: "us"})

	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Load(context.Background(), clicks.WorldStage))

	state, err := stateBatch(storage, 0, maxIndex)
	s.Require().NoError(err)
	s.Equal(map[uint32]string{10: "fr"}, state)
}

func (s *testSuite) TestALoadRefusesAMapWhoseRowsNameAnotherBoard() {
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{1: "fr"})
	persistence.Remapped(clicks.World, clicks.TestBoard+1)

	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	err := storage.Load(context.Background(), clicks.WorldStage)
	s.Require().ErrorIs(err, inmemory_tile_storage.ErrOtherBoard, "ids of another board would land on other ground")
	owner, _ := storage.Owner(1)
	s.Empty(owner, "nothing of it is played")
}

func (s *testSuite) TestAFlushNamesTheBoardTheMapPlays() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Load(ctx, clicks.World.On(clicks.TestBoard+1)))

	s.Require().NoError(storage.Set(ctx, 1, "fr"))
	s.Require().NoError(storage.Flush(ctx))

	s.Equal(clicks.TestBoard+1, persistence.BoardOn(clicks.World))
	s.Equal(clicks.World.On(clicks.TestBoard+1), storage.Stage())
}

func (s *testSuite) TestAFailedLoadRefusesTheBoot() {
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	persistence.FailWith(errors.New("connection refused"))

	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().ErrorContains(storage.Load(context.Background(), clicks.WorldStage), "connection refused")
}

func (s *testSuite) TestFlushWritesEachChangedTileOnceAsItIsNow() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().NoError(storage.Set(ctx, 1, "fr"))
	s.Require().NoError(storage.Set(ctx, 1, "us"))
	s.Require().NoError(storage.Set(ctx, 70, "fr"))
	s.Require().NoError(storage.Set(ctx, 2, "fr"))
	_, err := storage.Clear(ctx, clicks.Blast{Cleared: []uint32{2}})
	s.Require().NoError(err)

	s.Require().NoError(storage.Flush(ctx))

	s.Equal([][]inmemory_tile_storage.Tile{{{ID: 1, Owner: "us"}, {ID: 2}, {ID: 70, Owner: "fr"}}}, persistence.Saves())
	s.Equal(map[uint32]string{1: "us", 70: "fr"}, persistence.Stored())
}

func (s *testSuite) TestShieldsAreKeptWithTheirTileAndGoneWithItsFlag() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.loadedOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().NoError(storage.Set(ctx, 1, "fr"))
	s.Require().NoError(storage.Set(ctx, 2, "fr"))
	for _, tile := range []uint32{1, 1, 2} {
		_, err := storage.Shield(ctx, tile, "fr", 10, clicks.PlacementID{})
		s.Require().NoError(err)
	}
	s.Require().NoError(storage.Set(ctx, 2, "de"))
	s.Require().NoError(storage.Flush(ctx))

	s.Equal(map[uint32]int{1: 2}, persistence.StoredShields(), "a tile that changed hands lost its shields")

	restarted := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(restarted.Load(ctx, clicks.WorldStage))
	s.Equal(2, restarted.Shields(1))
	s.Zero(restarted.Shields(2))
}

func (s *testSuite) TestFlushWithNothingChangedWritesNothing() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().NoError(storage.Set(ctx, 1, "fr"))
	s.Require().NoError(storage.Flush(ctx))
	s.Require().NoError(storage.Flush(ctx))
	s.Require().NoError(storage.Set(ctx, 1, "fr"))
	s.Require().NoError(storage.Flush(ctx))

	s.Len(persistence.Saves(), 1, "a click on a tile already held changes nothing to write")
}

func (s *testSuite) TestAFailedFlushKeepsTheTilesForTheNextOne() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().NoError(storage.Set(ctx, 1, "fr"))
	persistence.FailWith(errors.New("connection reset"))
	s.Require().Error(storage.Flush(ctx))

	persistence.Heal()
	s.Require().NoError(storage.Set(ctx, 2, "us"))
	s.Require().NoError(storage.Flush(ctx))

	s.Equal(map[uint32]string{1: "fr", 2: "us"}, persistence.Stored())
}

func (s *testSuite) TestRestoredTilesAreFlushed() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)

	s.Require().NoError(storage.Set(ctx, 3, "ps"))
	s.Require().NoError(storage.Flush(ctx))
	_, err := storage.Restore(ctx, []clicks.Restoration{{Tile: 3, From: "ps", To: ""}})
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))

	s.Empty(persistence.Stored())
}

func (s *testSuite) TestRunFlushesOnShutdown() {
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{FlushInterval: time.Hour}, persistence)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		storage.Run(ctx)
	}()

	s.Require().NoError(storage.Set(context.Background(), 7, "fr"))
	cancel()

	s.Require().Eventually(func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, 5*time.Second, 10*time.Millisecond, "Run did not return after the context was cancelled")

	s.Equal(map[uint32]string{7: "fr"}, persistence.Stored())
}

func (s *testSuite) TestRunFlushesOnItsInterval() {
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{FlushInterval: 10 * time.Millisecond}, persistence)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go storage.Run(ctx)

	s.Require().NoError(storage.Set(context.Background(), 7, "fr"))

	s.Eventually(func() bool {
		return persistence.Stored()[7] == "fr"
	}, 5*time.Second, 10*time.Millisecond)
}

func (s *testSuite) TestAFortifiedProvinceStaysLockedAcrossARestart() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.loadedOn(inmemory_tile_storage.Config{}, persistence)
	for _, tile := range island {
		s.Require().NoError(storage.Set(ctx, tile, "fr"))
	}
	_, err := storage.Fortify(ctx, 90_003, "fr", 10)
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))

	restarted := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(restarted.Load(ctx, clicks.WorldStage))
	s.Require().NoError(restarted.Set(ctx, 90_001, "de"))
	s.Require().NoError(restarted.Set(ctx, 90_001, "fr"))
	_, err = restarted.Fortify(ctx, 90_001, "fr", 10)

	s.Require().ErrorIs(err, clicks.ErrFortifiedAlready)
	s.Equal(map[clicks.ProvinceID]string{1: "fr"}, persistence.StoredProvinces("test-provinces"))
	s.Equal(1, restarted.Shields(90_002))
}

func (s *testSuite) TestAWholeProvinceNobodyFortifiedIsLockedToItsHolderAtLoad() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{90_001: "fr", 90_002: "fr", 90_003: "fr"})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Load(ctx, clicks.WorldStage))
	s.Require().NoError(storage.Flush(ctx))

	_, err := storage.Fortify(ctx, 90_001, "fr", 10)

	s.Require().ErrorIs(err, clicks.ErrFortifiedAlready)
	s.Equal(map[clicks.ProvinceID]string{1: "fr"}, persistence.StoredProvinces("test-provinces"))
	s.Zero(storage.Shields(90_001))
}

func (s *testSuite) TestALockKeptForAnotherMapIsIgnored() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{90_001: "de", 90_002: "fr", 90_003: "fr"})
	persistence.Fortified("provinces-older.bin", 1, "fr")
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Load(ctx, clicks.WorldStage))
	s.Require().NoError(storage.Set(ctx, 90_001, "fr"))

	_, err := storage.Fortify(ctx, 90_001, "fr", 10)

	s.Require().NoError(err)
}

// The deploy that moved fortify onto provinces dropped every landmass's lock (docs/provinces.md): its boot finds no lock,
// and settles each province one flag holds whole to that flag. No shield is raised, none is taken away, and no take of
// a province settled so is a fortify.
func (s *testSuite) TestTheBootAfterTheLocksMovedToProvincesSettlesEachWholeProvinceWithNoShield() {
	ctx := context.Background()
	coast := []uint32{90_001, 90_002, 90_003, 90_004, 90_005}
	borders := clicks.BordersOf(maxIndex, coast)
	ground := clicks.GroundOfProvinces("earth-2", borders, clicks.NewCapitals(map[string]uint32{}),
		clicks.ProvincesOf("provinces-286f4c3e.bin", borders, coast[:3], coast[3:]))
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{
		90_001: "fr", 90_002: "fr", 90_003: "fr", 90_004: "fr", 90_005: "de",
	})
	for _, tile := range coast[:4] {
		persistence.Shield(tile, 2)
	}
	storage := inmemory_tile_storage.New(clicks.OneGround{Ground: ground}, clicks.NewPlayed(), clicks.NoCapitals(),
		inmemory_tile_storage.Config{}, persistence, cpatomic.FakeTransactioner{}, &inmemory_tile_storage.RecordedPlacements{},
		cptime.SystemClock{}, slog.New(slog.DiscardHandler))

	s.Require().NoError(storage.Load(ctx, clicks.WorldStage))
	s.Require().NoError(storage.Flush(ctx))

	s.Equal(map[clicks.ProvinceID]string{1: "fr"}, storage.Fortresses(), "the province France holds whole, not the coast")
	s.Equal(map[clicks.ProvinceID]string{1: "fr"}, persistence.StoredProvinces("provinces-286f4c3e.bin"))
	s.Equal(map[uint32]int{90_001: 2, 90_002: 2, 90_003: 2, 90_004: 2}, persistence.StoredShields(),
		"no shield raised, none taken away")
	s.Require().NoError(storage.Set(ctx, 90_001, "de"))
	s.Require().NoError(storage.Set(ctx, 90_001, "fr"))
	_, err := storage.Fortify(ctx, 90_001, "fr", 10)
	s.Require().ErrorIs(err, clicks.ErrFortifiedAlready, "no free fortify for France")
	s.Require().NoError(storage.Set(ctx, 90_004, "de"))
	_, err = storage.Fortify(ctx, 90_004, "de", 10)
	s.Require().NoError(err, "the province no flag held whole is fortified by the first to")
}

func (s *testSuite) TestAHolderIsFlushedWithItsTileAndReadBackAtBoot() {
	ana := cpsession.AccountID(uuid.New())
	asAna := cpctx.AddAccountToContext(context.Background(), ana.String())
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.loadedOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Click(asAna, island[0], "fr"))
	s.Require().NoError(storage.Set(context.Background(), island[1], "fr"))

	s.Require().NoError(storage.Flush(context.Background()))
	s.Equal(map[uint32]cpsession.AccountID{island[0]: ana}, persistence.StoredHolders())

	booted := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(booted.Load(context.Background(), clicks.WorldStage))
	s.Equal([]clicks.Holding{{Account: ana, Flag: "fr", Tiles: 1}}, booted.HoldingsOf(1))
}

func (s *testSuite) TestEachTileAskedIsReadWithItsFlagAndItsHolderInTheOrderAsked() {
	ana := cpsession.AccountID(uuid.New())
	storage := s.newStorage(inmemory_tile_storage.Config{})
	s.Require().NoError(storage.Click(cpctx.AddAccountToContext(context.Background(), ana.String()), island[0], "fr"))
	s.Require().NoError(storage.Set(context.Background(), island[1], "de"))

	s.Equal([]clicks.TileHold{
		{Tile: island[2], Holder: cpsession.NoAccount},
		{Tile: island[0], Flag: "fr", Holder: ana},
		{Tile: island[1], Flag: "de", Holder: cpsession.NoAccount},
		{Tile: 0, Holder: cpsession.NoAccount},
		{Tile: maxIndex + 1, Holder: cpsession.NoAccount},
	}, storage.HoldsOf([]uint32{island[2], island[0], island[1], 0, maxIndex + 1}),
		"an operator's write holds nobody, and a tile off the map is nobody's")
}

func (s *testSuite) TestAForgottenHolderIsFlushedAsNobody() {
	ana := cpsession.AccountID(uuid.New())
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Click(cpctx.AddAccountToContext(context.Background(), ana.String()), island[0], "fr"))
	s.Require().NoError(storage.Flush(context.Background()))

	_, err := storage.ForgetHolder(context.Background(), ana)
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(context.Background()))

	s.Empty(persistence.StoredHolders())
	s.Equal(map[uint32]string{island[0]: "fr"}, persistence.Stored())
}
