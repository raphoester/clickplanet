package inmemory_tile_storage_test

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

func tenShields() int { return 10 }

// An arena whose capitals start with no shield: France holds Paris (island[0]) and two tiles more, Germany Berlin (20).
func (s *testSuite) arenaWithFrance(persistence *inmemory_tile_storage.MemoryPersistence) *inmemory_tile_storage.Storage {
	ctx := context.Background()
	storage := s.withCapitals(persistence)
	_, err := storage.Blank(ctx, battle, capitalsWith(0), keeps(true))
	s.Require().NoError(err)
	s.Require().NoError(storage.Set(ctx, island[1], "fr"))
	s.Require().NoError(storage.Set(ctx, 5, "fr"))
	return storage
}

func (s *testSuite) TestATakeOfACapitalKnocksItsFlagOutInOneChangeAfterTheTake() {
	ctx := context.Background()
	storage := s.arenaWithFrance(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	changes, before, err := storage.Subscribe(ctx)
	s.Require().NoError(err)
	ana := uuid.NewString()

	impact, err := clicks.NewClaiming(storage, tenShields).Click(cpctx.AddAccountToContext(ctx, ana), island[0], "de")

	s.Require().NoError(err)
	s.Require().NotNil(impact.KnockedOut)
	s.Equal("fr", impact.KnockedOut.Country)
	s.Equal("de", impact.KnockedOut.By)
	s.Equal(island[0], impact.KnockedOut.Tile)
	s.Equal(ana, impact.KnockedOut.Account)
	s.Equal([]uint32{5, island[1]}, impact.KnockedOut.Cleared)

	take := <-changes
	s.Require().NotNil(take.Update)
	s.Equal(island[0], take.Update.Tile)
	knockOut := <-changes
	s.Require().NotNil(knockOut.KnockOut, "one change for every tile it cleared")
	s.Equal(before.Seq+2, knockOut.Version.Seq)
	s.Equal(*impact.KnockedOut, *knockOut.KnockOut)

	owner, _ := storage.Owner(island[0])
	s.Equal("de", owner, "the taker keeps the capital")
	for _, tile := range []uint32{5, island[1]} {
		owner, _ := storage.Owner(tile)
		s.Empty(owner, "tile %d", tile)
	}
	s.Zero(storage.Held("fr"))
}

func (s *testSuite) TestAFlagOutStandsNoMoreAndAMapReadSaysSo() {
	ctx := context.Background()
	storage := s.arenaWithFrance(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	s.True(storage.Stands("fr"))
	s.False(storage.Stands("gb-eng"), "a flag with no capital never stands in an arena")

	_, err := clicks.NewClaiming(storage, tenShields).Click(ctx, island[0], "de")
	s.Require().NoError(err)

	s.False(storage.Stands("fr"))
	s.True(storage.Stands("de"))
	batch, err := storage.StateBatchDense(1, 100)
	s.Require().NoError(err)
	s.Equal([]string{"fr"}, batch.KnockedOut)
}

func (s *testSuite) TestAFlagIsKnockedOutOnce() {
	ctx := context.Background()
	storage := s.arenaWithFrance(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	claiming := clicks.NewClaiming(storage, tenShields)
	_, err := claiming.Click(ctx, island[0], "de")
	s.Require().NoError(err)

	impact, err := claiming.Click(ctx, island[0], "it")

	s.Require().NoError(err)
	s.Equal(clicks.Taken, impact.Outcome)
	s.Nil(impact.KnockedOut, "Paris fell already")
	s.True(storage.Stands("de"), "losing another flag's capital knocks nobody out")
}

func (s *testSuite) TestACapitalLeftBlankStillFallsToTheNextTake() {
	ctx := context.Background()
	storage := s.arenaWithFrance(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	_, err := storage.Clear(ctx, clicks.Blast{Tile: island[0], Cleared: []uint32{island[0]}})
	s.Require().NoError(err)

	impact, err := clicks.NewClaiming(storage, tenShields).Claim(ctx, island[0], "de")

	s.Require().NoError(err)
	s.Require().NotNil(impact.KnockedOut)
	s.Equal([]uint32{5, island[1]}, impact.KnockedOut.Cleared)
}

func (s *testSuite) TestATakeKnocksNobodyOutWhileTheKnockOutDoesNotPlay() {
	ctx := context.Background()
	storage := s.withCapitalRules(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}), clicks.CapitalRules{})
	_, err := storage.Blank(ctx, battle, capitalsWith(0), keeps(true))
	s.Require().NoError(err)
	s.Require().NoError(storage.Set(ctx, 5, "fr"))

	impact, err := clicks.NewClaiming(storage, tenShields).Click(ctx, island[0], "de")

	s.Require().NoError(err)
	s.Nil(impact.KnockedOut, "an arena alone knocks nobody out")
	owner, _ := storage.Owner(5)
	s.Equal("fr", owner)
	s.True(storage.Stands("fr"))
	s.True(storage.Stands("gb-eng"), "every flag stands")
}

func (s *testSuite) TestTheKnockOutPlaysOnTheWorldsMapToo() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{island[0]: "fr", 5: "fr"})
	storage := s.withCapitals(persistence)

	impact, err := clicks.NewClaiming(storage, tenShields).Click(ctx, island[0], "de")

	s.Require().NoError(err)
	s.Require().NotNil(impact.KnockedOut)
	s.Equal([]uint32{5}, impact.KnockedOut.Cleared)
	s.False(storage.Stands("fr"))
	s.Require().NoError(storage.Flush(ctx))
	s.Contains(persistence.StoredOutFlagsOn(clicks.World), "fr")
}

func (s *testSuite) TestAnEnclosureKnocksOutOnceItsPocketIsWritten() {
	ctx := context.Background()
	storage := s.arenaWithFrance(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	s.Require().NoError(storage.Set(ctx, 7, "fr"))
	_, err := storage.Shield(ctx, 7, "fr", 10, clicks.PlacementID{})
	s.Require().NoError(err)
	changes, _, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	impacts, err := clicks.NewClaiming(storage, tenShields).Enclose(ctx, clicks.Enclosure{Country: "de"},
		[]uint32{island[0], island[1], 7})

	s.Require().NoError(err)
	s.Require().NotNil(impacts[0].KnockedOut)
	s.Equal([]uint32{5, 7}, impacts[0].KnockedOut.Cleared, "the pocket's takes stay, its struck tile goes blank")
	enclosure := <-changes
	s.Require().NotNil(enclosure.Enclosure)
	s.Equal([]uint32{island[0], island[1], 7}, enclosure.Enclosure.Filled)
	knockOut := <-changes
	s.Require().NotNil(knockOut.KnockOut, "after the enclosure")
	owner, _ := storage.Owner(island[1])
	s.Equal("de", owner)
	owner, _ = storage.Owner(7)
	s.Empty(owner)
	s.Zero(storage.Shields(7))
}

func (s *testSuite) TestAFlagOutIsSavedWithTheTilesAndReadAgainAtBoot() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.arenaWithFrance(persistence)
	_, err := clicks.NewClaiming(storage, tenShields).Click(ctx, island[0], "de")
	s.Require().NoError(err)

	persistence.FailWith(errors.New("postgres is restarting"))
	s.Require().Error(storage.Flush(ctx))
	persistence.Heal()
	s.Require().NoError(storage.Flush(ctx))

	out := persistence.StoredOutFlagsOn(battle)
	s.Require().Contains(out, "fr", "a failed flush keeps it for the next")
	s.Equal("de", out["fr"].By)
	s.Equal(map[uint32]string{island[0]: "de", 20: "de"}, persistence.StoredOn(battle))
	s.Empty(persistence.StoredOutFlagsOn(clicks.World))

	booted := s.withCapitals(persistence)
	s.Require().NoError(booted.Load(ctx, battle))
	s.False(booted.Stands("fr"))
	s.True(booted.Stands("de"))
}

func (s *testSuite) TestTheFlagsOutGoWithTheirArena() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.arenaWithFrance(persistence)
	_, err := clicks.NewClaiming(storage, tenShields).Click(ctx, island[0], "de")
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))

	_, err = storage.PutBack(ctx, battle, keeps(true))

	s.Require().NoError(err)
	s.Empty(persistence.StoredOutFlagsOn(battle))
	s.True(storage.Stands("fr"))
	batch, err := storage.StateBatchDense(1, 100)
	s.Require().NoError(err)
	s.Empty(batch.KnockedOut)
}
