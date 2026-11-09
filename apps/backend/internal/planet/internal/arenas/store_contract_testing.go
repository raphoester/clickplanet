//go:build testing

package arenas

import (
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var (
	contractOpenedAt = time.Date(2026, 11, 10, 21, 0, 3, 123456000, time.UTC)
	contractEndsAt   = time.Date(2026, 11, 10, 23, 0, 0, 0, time.UTC)
	contractAna      = cpsession.AccountID(uuid.MustParse("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"))
	contractBefore   = clicks.MapCopy{
		Tiles: []clicks.TileCopy{
			{ID: 1, Owner: "fr", Shields: 3, Holder: contractAna},
			{ID: 2, Owner: "fr"},
			{ID: 250_000, Owner: "de", Shields: 255},
		},
		Landmasses: []clicks.LandmassCopy{{ID: 4, Asset: "borders-v2.bin", FortifiedBy: "fr"}},
	}
)

func (s *StoreContractSuite) opening(id ID) Opening {
	return Opening{ID: id, EndsAt: contractEndsAt, OpenedAt: contractOpenedAt}
}

func (s *StoreContractSuite) find(id ID) Arena {
	arena, err := s.store.Find(s.T().Context(), id)
	s.Require().NoError(err)
	return arena
}

func (s *StoreContractSuite) sameOpening(want, got Opening) {
	s.Equal(want.ID, got.ID)
	s.True(want.EndsAt.Equal(got.EndsAt), "ends at: want %s, got %s", want.EndsAt, got.EndsAt)
	s.True(want.OpenedAt.Equal(got.OpenedAt), "opened at: want %s, got %s", want.OpenedAt, got.OpenedAt)
}

func (s *StoreContractSuite) TestAnArenaNeverKeptNeverOpened() {
	s.Equal(Arena{ID: "battle"}, s.find("battle"))
	playing, err := s.store.Playing(s.T().Context())
	s.Require().NoError(err)
	s.False(playing.Opened)
}

func (s *StoreContractSuite) TestAnArenaOpenIsPlayingWithItsOpening() {
	s.Require().NoError(s.store.Open(s.T().Context(), s.opening("battle"), contractBefore))

	found := s.find("battle")
	s.True(found.Opened)
	s.False(found.Closed)
	s.sameOpening(s.opening("battle"), found.Opening)
	playing, err := s.store.Playing(s.T().Context())
	s.Require().NoError(err)
	s.True(playing.Opened)
	s.sameOpening(s.opening("battle"), playing.Opening)
}

func (s *StoreContractSuite) TestAClosedArenaAnswersTheMapFromBeforeItOnce() {
	ctx := s.T().Context()
	s.Require().NoError(s.store.Open(ctx, s.opening("battle"), contractBefore))

	before, err := s.store.Close(ctx, Closing{ID: "battle", ClosedAt: contractEndsAt.Add(3 * time.Second)})

	s.Require().NoError(err)
	s.ElementsMatch(contractBefore.Tiles, before.Tiles)
	s.Equal(contractBefore.Landmasses, before.Landmasses)
	found := s.find("battle")
	s.True(found.Opened)
	s.True(found.Closed)
	playing, err := s.store.Playing(ctx)
	s.Require().NoError(err)
	s.False(playing.Opened, "no arena is open once it closed")
}

func (s *StoreContractSuite) TestAnArenaNeverOpenedClosesWithNoMap() {
	before, err := s.store.Close(s.T().Context(), Closing{ID: "battle", ClosedAt: contractEndsAt})

	s.Require().NoError(err)
	s.Empty(before.Tiles)
	s.Empty(before.Landmasses)
	s.Equal(Arena{ID: "battle", Closed: true}, s.find("battle"))
}

func (s *StoreContractSuite) TestADeletedAccountHoldsNoTileOfAnArena() {
	ctx := s.T().Context()
	s.Require().NoError(s.store.Open(ctx, s.opening("battle"), contractBefore))

	s.Require().NoError(s.store.ForgetHolder(ctx, contractAna))

	before, err := s.store.Close(ctx, Closing{ID: "battle", ClosedAt: contractEndsAt})
	s.Require().NoError(err)
	s.Contains(before.Tiles, clicks.TileCopy{ID: 1, Owner: "fr", Shields: 3})
}
