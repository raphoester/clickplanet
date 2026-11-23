package postgres_tile_store_test

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var setter = cpsession.AccountID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x6f}

func (s *testSuite) trapsOn(played clicks.MapID) []clicks.TrapSnapshot {
	var traps []clicks.TrapSnapshot
	s.Require().NoError(s.store.LoadTraps(context.Background(), played, func(trap clicks.TrapSnapshot) {
		traps = append(traps, trap)
	}))
	return traps
}

func trapOn(tile uint32, flag string) clicks.TrapSnapshot {
	return clicks.TrapSnapshot{Tile: tile, Flag: flag, Account: setter}
}

func (s *testSuite) TestEachTrapLaidIsKeptAndEachOneGoneIsDeleted() {
	ctx := context.Background()
	s.Require().NoError(s.store.SaveTraps(ctx, clicks.WorldStage, []clicks.TrapSnapshot{trapOn(9, "fr"), trapOn(5, "fr")}, nil))

	s.Require().NoError(s.store.SaveTraps(ctx, clicks.WorldStage, []clicks.TrapSnapshot{trapOn(7, "de")}, []uint32{9, 11}))

	s.Equal([]clicks.TrapSnapshot{trapOn(5, "fr"), trapOn(7, "de")}, s.trapsOn(clicks.World), "in tile order")
	s.Empty(s.trapsOn(battle))
}

func (s *testSuite) TestATrapLaidAgainOnATileReplacesTheOneKept() {
	ctx := context.Background()
	s.Require().NoError(s.store.SaveTraps(ctx, clicks.WorldStage, []clicks.TrapSnapshot{trapOn(5, "fr")}, nil))

	s.Require().NoError(s.store.SaveTraps(ctx, clicks.WorldStage, []clicks.TrapSnapshot{trapOn(5, "de")}, nil))

	s.Equal([]clicks.TrapSnapshot{trapOn(5, "de")}, s.trapsOn(clicks.World))
}

func (s *testSuite) TestAWipeDeletesTheTrapsOfItsMapAlone() {
	ctx := context.Background()
	s.Require().NoError(s.store.SaveTraps(ctx, clicks.WorldStage, []clicks.TrapSnapshot{trapOn(5, "fr")}, nil))
	s.Require().NoError(s.store.SaveTraps(ctx, battle.On(clicks.TestBoard), []clicks.TrapSnapshot{trapOn(9, "de")}, nil))

	_, err := s.store.Wipe(ctx, battle, func(context.Context) (bool, error) { return true, nil })
	s.Require().NoError(err)

	s.Empty(s.trapsOn(battle))
	s.Equal([]clicks.TrapSnapshot{trapOn(5, "fr")}, s.trapsOn(clicks.World))
}

func (s *testSuite) TestAForgottenHoldersTrapsAreDeletedFromEveryMap() {
	ctx := context.Background()
	other := trapOn(7, "de")
	other.Account = cpsession.AccountID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x99}
	s.Require().NoError(s.store.SaveTraps(ctx, clicks.WorldStage, []clicks.TrapSnapshot{trapOn(5, "fr"), other}, nil))
	s.Require().NoError(s.store.SaveTraps(ctx, battle.On(clicks.TestBoard), []clicks.TrapSnapshot{trapOn(9, "de")}, nil))

	s.Require().NoError(s.store.ForgetHolder(ctx, setter))

	s.Equal([]clicks.TrapSnapshot{other}, s.trapsOn(clicks.World))
	s.Empty(s.trapsOn(battle))
}
