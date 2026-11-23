package inmemory_tile_storage_test

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// A corridor, 1 - 2 - 3 - 4 - 5 - 6, France's capital on 1, and Germany's on 20, apart.
var (
	corridor         = []clicks.Edge{{From: 1, To: 2}, {From: 2, To: 3}, {From: 3, To: 4}, {From: 4, To: 5}, {From: 5, To: 6}}
	corridorCapitals = clicks.NewCapitals(map[string]uint32{"fr": 1, "de": 20})
)

func franceInTheCorridor() *inmemory_tile_storage.MemoryPersistence {
	return inmemory_tile_storage.NewMemoryPersistence(
		map[uint32]string{1: "fr", 2: "fr", 3: "fr", 4: "fr", 5: "fr", 6: "fr", 20: "de"},
	)
}

func (s *testSuite) inTheCorridor(
	persistence *inmemory_tile_storage.MemoryPersistence, played clicks.CapitalRules,
) *inmemory_tile_storage.Storage {
	play := clicks.NewCapitalPlay(
		func() *clicks.Capitals { return corridorCapitals }, func() clicks.CapitalRules { return played },
	)
	ground := clicks.LinkedGround(clicks.BordersOf(30), corridorCapitals, corridor)
	storage := inmemory_tile_storage.New(clicks.OneGround{Ground: ground}, clicks.NewPlayed(), play,
		inmemory_tile_storage.Config{}, persistence, cpatomic.FakeTransactioner{}, &inmemory_tile_storage.RecordedPlacements{},
		cptime.SystemClock{}, slog.New(slog.DiscardHandler))
	s.Require().NoError(storage.Load(context.Background(), clicks.WorldStage))
	return storage
}

func (s *testSuite) TestATakeThatCutsLandFromItsCapitalClearsItInOneChangeAfterTheTake() {
	ctx := context.Background()
	persistence := franceInTheCorridor()
	storage := s.inTheCorridor(persistence, clicks.CapitalRules{CutOff: true})
	_, err := storage.Shield(ctx, 5, "fr", 10, clicks.PlacementID{})
	s.Require().NoError(err)
	changes, before, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	impact, err := clicks.NewClaiming(storage, tenShields, clicks.NewSnareLog()).Click(cpctx.AddAccountToContext(ctx, uuid.NewString()), 3, "de")

	s.Require().NoError(err)
	s.Require().NotNil(impact.CutOff)
	s.Equal(clicks.CutOff{Country: "fr", By: "de", Tile: 3, Cleared: []uint32{4, 5, 6}}, *impact.CutOff)

	take := <-changes
	s.Require().NotNil(take.Update)
	s.Equal(uint32(3), take.Update.Tile)
	cutOff := <-changes
	s.Require().NotNil(cutOff.CutOff, "one change for every tile it cleared")
	s.Equal(before.Seq+2, cutOff.Version.Seq)
	s.Equal(*impact.CutOff, *cutOff.CutOff)

	for _, tile := range []uint32{4, 5, 6} {
		owner, _ := storage.Owner(tile)
		s.Empty(owner, "tile %d", tile)
	}
	s.Zero(storage.Shields(5), "a shield does not hold a tile cut off")
	s.Equal(2, storage.Held("fr"))
	s.Require().NoError(storage.Flush(ctx))
	s.Equal(map[uint32]string{1: "fr", 2: "fr", 3: "de", 20: "de"}, persistence.Stored())
}

func (s *testSuite) TestATakeAtTheTipOfTheLandCutsNothing() {
	storage := s.inTheCorridor(franceInTheCorridor(), clicks.CapitalRules{CutOff: true})

	impact, err := clicks.NewClaiming(storage, tenShields, clicks.NewSnareLog()).Click(context.Background(), 6, "de")

	s.Require().NoError(err)
	s.Equal(clicks.Taken, impact.Outcome)
	s.Nil(impact.CutOff)
	s.Equal(5, storage.Held("fr"))
}

func (s *testSuite) TestATakeCutsNothingWhileTheCutOffDoesNotPlay() {
	storage := s.inTheCorridor(franceInTheCorridor(), clicks.CapitalRules{KnockOut: true})

	impact, err := clicks.NewClaiming(storage, tenShields, clicks.NewSnareLog()).Click(context.Background(), 3, "de")

	s.Require().NoError(err)
	s.Nil(impact.CutOff)
	owner, _ := storage.Owner(6)
	s.Equal("fr", owner)
}

func (s *testSuite) TestACapitalTakenKnocksItsFlagOutAndCutsNothingMore() {
	storage := s.inTheCorridor(franceInTheCorridor(), clicks.CapitalRules{KnockOut: true, CutOff: true})

	impact, err := clicks.NewClaiming(storage, tenShields, clicks.NewSnareLog()).Click(context.Background(), 1, "de")

	s.Require().NoError(err)
	s.Require().NotNil(impact.KnockedOut)
	s.Equal([]uint32{2, 3, 4, 5, 6}, impact.KnockedOut.Cleared)
	s.Nil(impact.CutOff, "the knock-out left nothing to cut")
}

func (s *testSuite) TestACapitalTakenWithNoKnockOutCutsTheWholeLand() {
	storage := s.inTheCorridor(franceInTheCorridor(), clicks.CapitalRules{CutOff: true})

	impact, err := clicks.NewClaiming(storage, tenShields, clicks.NewSnareLog()).Click(context.Background(), 1, "de")

	s.Require().NoError(err)
	s.Nil(impact.KnockedOut)
	s.Require().NotNil(impact.CutOff)
	s.Equal([]uint32{2, 3, 4, 5, 6}, impact.CutOff.Cleared)
	s.Zero(storage.Held("fr"))
	s.True(storage.Stands("fr"), "a cut-off knocks nobody out")
}

func (s *testSuite) TestAnEnclosureCutsOnceItsPocketIsWritten() {
	ctx := context.Background()
	storage := s.inTheCorridor(franceInTheCorridor(), clicks.CapitalRules{CutOff: true})
	changes, _, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	impacts, err := clicks.NewClaiming(storage, tenShields, clicks.NewSnareLog()).Enclose(ctx, clicks.Enclosure{Country: "de"}, []uint32{3, 4})

	s.Require().NoError(err)
	s.Require().Len(impacts, 2)
	s.Nil(impacts[0].CutOff)
	s.Require().NotNil(impacts[1].CutOff, "told on the flag's last tile taken")
	s.Equal(clicks.CutOff{Country: "fr", By: "de", Tile: 4, Cleared: []uint32{5, 6}}, *impacts[1].CutOff)
	enclosure := <-changes
	s.Require().NotNil(enclosure.Enclosure)
	cutOff := <-changes
	s.Require().NotNil(cutOff.CutOff)
	s.Equal([]uint32{5, 6}, cutOff.CutOff.Cleared)
}
