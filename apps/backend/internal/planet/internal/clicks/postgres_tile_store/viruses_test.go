package postgres_tile_store_test

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func (s *testSuite) virusesOn(played clicks.MapID) []clicks.VirusSnapshot {
	var viruses []clicks.VirusSnapshot
	s.Require().NoError(s.store.LoadViruses(context.Background(), played, func(virus clicks.VirusSnapshot) {
		viruses = append(viruses, virus)
	}))
	return viruses
}

func virusOf(id clicks.VirusID, infected ...uint32) clicks.VirusSnapshot {
	return clicks.VirusSnapshot{
		ID:      id,
		Account: cpsession.AccountID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x5f},
		Scope:   "2001:db8::/64", Flag: "fr", Victim: "de", Tile: infected[0], Infected: infected, Infections: 5,
		Next: time.Date(2026, 11, 16, 12, 0, 20, 0, time.UTC),
	}
}

func (s *testSuite) TestTheLiveVirusesOfAMapAreKeptAsAWholeAndReadBack() {
	ctx := context.Background()
	first := virusOf("0192f5a0-7c00-7000-8000-000000000001", 5, 4, 6)
	first.Linked = true
	second := virusOf("0192f5a0-7c00-7000-8000-000000000002", 9)

	s.Require().NoError(s.store.SaveViruses(ctx, clicks.WorldStage, []clicks.VirusSnapshot{first, second}))
	s.Require().NoError(s.store.SaveViruses(ctx, clicks.WorldStage, []clicks.VirusSnapshot{first}),
		"a virus the next save leaves out ended")

	s.Equal([]clicks.VirusSnapshot{first}, s.virusesOn(clicks.World))
	s.Empty(s.virusesOn(battle))
}

func (s *testSuite) TestAWipeDeletesTheVirusesOfItsMapAlone() {
	ctx := context.Background()
	world := virusOf("0192f5a0-7c00-7000-8000-000000000001", 5)
	s.Require().NoError(s.store.SaveViruses(ctx, clicks.WorldStage, []clicks.VirusSnapshot{world}))
	s.Require().NoError(s.store.SaveViruses(ctx, battle.On(clicks.TestBoard), []clicks.VirusSnapshot{virusOf("0192f5a0-7c00-7000-8000-000000000002", 9)}))

	_, err := s.store.Wipe(ctx, battle, func(context.Context) (bool, error) { return true, nil })
	s.Require().NoError(err)

	s.Empty(s.virusesOn(battle))
	s.Equal([]clicks.VirusSnapshot{world}, s.virusesOn(clicks.World))
}
