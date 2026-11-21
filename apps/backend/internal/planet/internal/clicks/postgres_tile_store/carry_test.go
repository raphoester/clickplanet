package postgres_tile_store_test

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras/postgres_era_store"
)

var earth2 = clicks.World.On(eras.Earth2.ID)

// The world on earth-1 as a switch finds it: tiles, a lock, a mine, a virus and a flag out.
func (s *testSuite) worldBeforeASwitch() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, clicks.WorldStage, owned("fr", 1, 2, 3),
		[]inmemory_tile_storage.Province{{ID: 1, Asset: "provinces.bin", FortifiedBy: "fr"}},
		[]clicks.OutFlag{{Flag: "pl", By: "de", At: time.Date(2026, 11, 21, 20, 0, 0, 0, time.UTC)}}))
	s.Require().NoError(s.store.SaveMines(ctx, clicks.WorldStage, []clicks.MineSnapshot{mineOn(1, "fr")}, nil))
	s.Require().NoError(s.store.SaveViruses(ctx, clicks.WorldStage, []clicks.VirusSnapshot{
		virusOf("0192f5a0-7c00-7000-8000-000000000001", 1),
	}))
}

func (s *testSuite) TestACarryMovesTheMapOntoTheNewBoardInTheEraStartsTransaction() {
	ctx := context.Background()
	s.worldBeforeASwitch()
	kept := postgres_era_store.New(s.db)
	at := time.Date(2026, 11, 22, 4, 0, 0, 0, time.UTC)
	carried := append(owned("fr", 1, 2, 3, 4), owned("de", 900_000)...)
	carried[0].Shields = 3

	switched, err := s.store.Carry(ctx, earth2, carried, []clicks.MineSnapshot{mineOn(3, "fr")},
		func(ctx context.Context) (bool, error) { return true, kept.Start(ctx, earth2, at) })

	s.Require().NoError(err)
	s.True(switched)
	s.Equal(map[uint32]string{1: "fr", 2: "fr", 3: "fr", 4: "fr", 900_000: "de"}, s.load())
	boards, err := s.store.Boards(ctx, clicks.World)
	s.Require().NoError(err)
	s.Equal([]clicks.BoardID{eras.Earth2.ID}, boards, "every row names the new board: the boot plays it")
	var mines []clicks.MineSnapshot
	s.Require().NoError(s.store.LoadMines(ctx, clicks.World, func(mine clicks.MineSnapshot) { mines = append(mines, mine) }))
	s.Equal([]clicks.MineSnapshot{mineOn(3, "fr")}, mines)
	var viruses []clicks.VirusSnapshot
	s.Require().NoError(s.store.LoadViruses(ctx, clicks.World, func(virus clicks.VirusSnapshot) { viruses = append(viruses, virus) }))
	s.Empty(viruses, "live viruses end")
	locks := map[clicks.ProvinceID]string{}
	s.Require().NoError(s.store.LoadProvinces(ctx, clicks.World, "provinces.bin", func(id clicks.ProvinceID, by string) { locks[id] = by }))
	s.Empty(locks, "the old board's locks are dropped")
	var out []clicks.OutFlag
	s.Require().NoError(s.store.LoadOutFlags(ctx, clicks.World, func(flag clicks.OutFlag) { out = append(out, flag) }))
	s.Len(out, 1, "a flag out names no tile, and stays")
	era, err := kept.Latest(ctx, clicks.World)
	s.Require().NoError(err)
	s.Equal(eras.Earth2, era.Board)
	s.True(at.Equal(era.StartedAt))
}

func (s *testSuite) TestACarryThatFailsKeepsTheMapOnItsBoardAndNoEra() {
	ctx := context.Background()
	s.worldBeforeASwitch()
	kept := postgres_era_store.New(s.db)
	broken := errors.New("broken")

	_, err := s.store.Carry(ctx, earth2, owned("fr", 7), nil, func(ctx context.Context) (bool, error) {
		s.Require().NoError(kept.Start(ctx, earth2, time.Now()))
		return false, broken
	})

	s.Require().ErrorIs(err, broken)
	s.Equal(map[uint32]string{1: "fr", 2: "fr", 3: "fr"}, s.load())
	boards, err := s.store.Boards(ctx, clicks.World)
	s.Require().NoError(err)
	s.Equal([]clicks.BoardID{clicks.TestBoard}, boards)
	era, err := kept.Latest(ctx, clicks.World)
	s.Require().NoError(err)
	s.Equal(eras.Earth1, era.Board, "the era rolled back with the carry")
}
