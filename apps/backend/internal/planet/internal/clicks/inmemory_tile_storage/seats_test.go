package inmemory_tile_storage_test

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const rennes = "r-0000000001"

var seats = []clicks.Capital{{Country: rennes, Tile: 30}}

func seatedStart() []clicks.StartTile {
	return append(capitalsWith(50), clicks.StartTile{Tile: 30, Country: rennes, Shields: 50})
}

// withSeats reads the capitals of the map in play, its seats with them, and plays the knock-out and fixed shields.
func (s *testSuite) withSeats(persistence *inmemory_tile_storage.MemoryPersistence) (*inmemory_tile_storage.Storage, *clicks.Played) {
	played := clicks.NewPlayed()
	play := clicks.NewCapitalPlay(played.Capitals, func() clicks.CapitalRules {
		return clicks.CapitalRules{KnockOut: true, FixedShields: true}
	})
	storage := inmemory_tile_storage.New(clicks.OneGround{Ground: clicks.GroundWith(borders, capitals)}, played, play,
		inmemory_tile_storage.Config{}, persistence, cpatomic.FakeTransactioner{}, &inmemory_tile_storage.RecordedPlacements{},
		cptime.SystemClock{}, slog.New(slog.DiscardHandler))
	s.Require().NoError(storage.Load(context.Background(), clicks.WorldStage))
	return storage, played
}

func (s *testSuite) TestABlankPlaysItsSeatsAsCapitalsAndKeepsThemWithItsMap() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage, played := s.withSeats(persistence)

	_, err := storage.Blank(ctx, battle.On(clicks.TestBoard), seatedStart(), seats, opens(true))

	s.Require().NoError(err)
	seat, ok := played.Capitals().Of(rennes)
	s.True(ok)
	s.Equal(uint32(30), seat)
	s.True(storage.Stands(rennes))
	s.False(storage.Stands("r-0000000002"), "a flag with no seat has no capital")
	s.Equal(seats, persistence.StoredSeatsOn(battle))
	batch, err := storage.StateBatchDense(1, maxIndex)
	s.Require().NoError(err)
	s.Equal(seats, batch.Seats, "a map read says them")
	s.Equal(50, storage.Shields(30))
}

func (s *testSuite) TestNoShieldGoesOnASeat() {
	ctx := context.Background()
	storage, _ := s.withSeats(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	_, err := storage.Blank(ctx, battle.On(clicks.TestBoard), seatedStart(), seats, opens(true))
	s.Require().NoError(err)

	_, err = storage.Shield(ctx, 30, rennes, 255, clicks.PlacementID{})

	s.Require().ErrorIs(err, clicks.ErrCapital)
}

func (s *testSuite) TestATakeOfASeatKnocksItsFlagOut() {
	ctx := context.Background()
	storage, _ := s.withSeats(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	_, err := storage.Blank(ctx, battle.On(clicks.TestBoard), []clicks.StartTile{{Tile: 30, Country: rennes}}, seats,
		opens(true))
	s.Require().NoError(err)
	s.Require().NoError(storage.Set(ctx, 31, rennes))

	impact, err := clicks.NewClaiming(storage, tenShields, clicks.NewSnareLog()).Click(ctx, 30, "de")

	s.Require().NoError(err)
	s.Require().NotNil(impact.KnockedOut)
	s.Equal(rennes, impact.KnockedOut.Country)
	s.Equal([]uint32{31}, impact.KnockedOut.Cleared)
	s.False(storage.Stands(rennes))
}

func (s *testSuite) TestABootReadsTheSeatsWithTheArenasMap() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage, _ := s.withSeats(persistence)
	_, err := storage.Blank(ctx, battle.On(clicks.TestBoard), seatedStart(), seats, opens(true))
	s.Require().NoError(err)

	booted, played := s.withSeats(persistence)
	s.Require().NoError(booted.Load(ctx, battle.On(clicks.TestBoard)))

	s.True(played.Capitals().Has(rennes))
	s.True(booted.Stands(rennes))
}

func (s *testSuite) TestTheWorldPlaysAgainWithNoSeat() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage, played := s.withSeats(persistence)
	_, err := storage.Blank(ctx, battle.On(clicks.TestBoard), seatedStart(), seats, opens(true))
	s.Require().NoError(err)

	_, err = storage.PutBack(ctx, battle, clicks.WorldStage, keeps(true))

	s.Require().NoError(err)
	s.False(played.Capitals().Has(rennes))
	s.Empty(persistence.StoredSeatsOn(battle), "the wipe deleted them")
	batch, err := storage.StateBatchDense(1, maxIndex)
	s.Require().NoError(err)
	s.Empty(batch.Seats)
}
