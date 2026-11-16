package postgres_tile_store_test

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var sapper = cpsession.AccountID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x5f}

func (s *testSuite) minesOn(played clicks.MapID) []clicks.MineSnapshot {
	var mines []clicks.MineSnapshot
	s.Require().NoError(s.store.LoadMines(context.Background(), played, func(mine clicks.MineSnapshot) {
		mines = append(mines, mine)
	}))
	return mines
}

func mineOn(tile uint32, flag string) clicks.MineSnapshot {
	return clicks.MineSnapshot{Tile: tile, Flag: flag, Account: sapper, Scope: "2001:db8::/64"}
}

func (s *testSuite) TestEachMineLaidIsKeptAndEachOneGoneIsDeleted() {
	ctx := context.Background()
	s.Require().NoError(s.store.SaveMines(ctx, clicks.World, []clicks.MineSnapshot{mineOn(9, "fr"), mineOn(5, "fr")}, nil))

	s.Require().NoError(s.store.SaveMines(ctx, clicks.World, []clicks.MineSnapshot{mineOn(7, "de")}, []uint32{9, 11}))

	s.Equal([]clicks.MineSnapshot{mineOn(5, "fr"), mineOn(7, "de")}, s.minesOn(clicks.World), "in tile order")
	s.Empty(s.minesOn(battle))
}

func (s *testSuite) TestAMineLaidAgainOnATileReplacesTheOneKept() {
	ctx := context.Background()
	s.Require().NoError(s.store.SaveMines(ctx, clicks.World, []clicks.MineSnapshot{mineOn(5, "fr")}, nil))

	s.Require().NoError(s.store.SaveMines(ctx, clicks.World, []clicks.MineSnapshot{mineOn(5, "de")}, nil))

	s.Equal([]clicks.MineSnapshot{mineOn(5, "de")}, s.minesOn(clicks.World))
}

func (s *testSuite) TestAWipeDeletesTheMinesOfItsMapAlone() {
	ctx := context.Background()
	s.Require().NoError(s.store.SaveMines(ctx, clicks.World, []clicks.MineSnapshot{mineOn(5, "fr")}, nil))
	s.Require().NoError(s.store.SaveMines(ctx, battle, []clicks.MineSnapshot{mineOn(9, "de")}, nil))

	_, err := s.store.Wipe(ctx, battle, func(context.Context) (bool, error) { return true, nil })
	s.Require().NoError(err)

	s.Empty(s.minesOn(battle))
	s.Equal([]clicks.MineSnapshot{mineOn(5, "fr")}, s.minesOn(clicks.World))
}

func (s *testSuite) TestAForgottenHoldersMinesAreDeletedFromEveryMap() {
	ctx := context.Background()
	other := mineOn(7, "de")
	other.Account = cpsession.AccountID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x99}
	s.Require().NoError(s.store.SaveMines(ctx, clicks.World, []clicks.MineSnapshot{mineOn(5, "fr"), other}, nil))
	s.Require().NoError(s.store.SaveMines(ctx, battle, []clicks.MineSnapshot{mineOn(9, "de")}, nil))

	s.Require().NoError(s.store.ForgetHolder(ctx, sapper))

	s.Equal([]clicks.MineSnapshot{other}, s.minesOn(clicks.World))
	s.Empty(s.minesOn(battle))
}
