//go:build testing

package arenas

import (
	"time"

	"github.com/stretchr/testify/suite"
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
)

func (s *StoreContractSuite) opening(id ID) Opening {
	return Opening{ID: id, EndsAt: contractEndsAt, OpenedAt: contractOpenedAt, Freeze: 8500 * time.Millisecond}
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
	s.Equal(want.Freeze, got.Freeze)
}

func (s *StoreContractSuite) TestAnArenaNeverKeptNeverOpened() {
	s.Equal(Arena{ID: "battle"}, s.find("battle"))
	playing, err := s.store.Playing(s.T().Context())
	s.Require().NoError(err)
	s.False(playing.Opened)
}

func (s *StoreContractSuite) TestAnArenaOpenIsPlayingWithItsOpening() {
	s.Require().NoError(s.store.Open(s.T().Context(), s.opening("battle")))

	found := s.find("battle")
	s.True(found.Opened)
	s.False(found.Closed)
	s.sameOpening(s.opening("battle"), found.Opening)
	playing, err := s.store.Playing(s.T().Context())
	s.Require().NoError(err)
	s.True(playing.Opened)
	s.sameOpening(s.opening("battle"), playing.Opening)
}

func (s *StoreContractSuite) TestAClosedArenaPlaysNoMore() {
	ctx := s.T().Context()
	s.Require().NoError(s.store.Open(ctx, s.opening("battle")))

	s.Require().NoError(s.store.Close(ctx, Closing{ID: "battle", ClosedAt: contractEndsAt.Add(3 * time.Second)}))

	found := s.find("battle")
	s.True(found.Opened)
	s.True(found.Closed)
	playing, err := s.store.Playing(ctx)
	s.Require().NoError(err)
	s.False(playing.Opened, "no arena is open once it closed")
}

func (s *StoreContractSuite) TestAnArenaNeverOpenedCloses() {
	s.Require().NoError(s.store.Close(s.T().Context(), Closing{ID: "battle", ClosedAt: contractEndsAt}))

	s.Equal(Arena{ID: "battle", Closed: true}, s.find("battle"))
}
