//go:build testing

package eras

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// StoreContractSuite starts from what the migration that made eras keeps: earth-1, and the world on it since the
// epoch.
type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var (
	contractOpens  = time.Date(2026, 11, 14, 18, 0, 0, 500_000_000, time.UTC)
	contractReopen = time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
)

func (s *StoreContractSuite) TestTheShippedBoardsAreKept() {
	boards, err := s.store.Boards(s.T().Context())
	s.Require().NoError(err)
	s.Equal([]Board{Earth1, Earth2, Siege1}, boards)
}

func (s *StoreContractSuite) TestTheWorldPlaysEarth1SinceTheEpoch() {
	era, err := s.store.Latest(s.T().Context(), clicks.World)
	s.Require().NoError(err)
	s.Equal(Era{Map: clicks.World, Board: Earth1, StartedAt: Epoch}, s.utc(era))
}

func (s *StoreContractSuite) TestAMapWithNoEraIsNamedSo() {
	_, err := s.store.Latest(s.T().Context(), "battle")
	s.ErrorIs(err, ErrNoEra)
}

func (s *StoreContractSuite) TestAnEraStartedIsTheLatestAndKeptOnce() {
	s.Require().NoError(s.store.Start(s.T().Context(), clicks.MapID("battle").On(Earth1.ID), contractOpens))
	s.Require().NoError(s.store.Start(s.T().Context(), clicks.MapID("battle").On(Earth1.ID), contractOpens))

	era, err := s.store.Latest(s.T().Context(), "battle")
	s.Require().NoError(err)
	s.Equal(Era{Map: "battle", Board: Earth1, StartedAt: contractOpens}, s.utc(era))

	covering, err := s.store.Covering(s.T().Context(), "battle", contractOpens.Add(-time.Hour), contractOpens.Add(time.Hour))
	s.Require().NoError(err)
	s.Len(covering, 1, "kept once")
}

func (s *StoreContractSuite) TestAnEraOnABoardNotKeptIsRefused() {
	s.Error(s.store.Start(s.T().Context(), clicks.MapID("battle").On(Fixture.ID), contractOpens), "no migration keeps the fixture")
}

func (s *StoreContractSuite) TestAWindowIsCoveredByTheEraInForceAtItsStartAndEachStartedInIt() {
	ctx := s.T().Context()
	s.Require().NoError(s.store.Start(ctx, clicks.World.On(Earth1.ID), contractOpens))
	s.Require().NoError(s.store.Start(ctx, clicks.World.On(Earth1.ID), contractReopen))

	covering, err := s.store.Covering(ctx, clicks.World, contractOpens.Add(time.Hour), contractReopen.Add(time.Hour))
	s.Require().NoError(err)
	s.Equal([]Era{
		{Map: clicks.World, Board: Earth1, StartedAt: contractOpens},
		{Map: clicks.World, Board: Earth1, StartedAt: contractReopen},
	}, s.utcAll(covering))

	covering, err = s.store.Covering(ctx, clicks.World, contractOpens, contractReopen)
	s.Require().NoError(err)
	s.Equal([]Era{{Map: clicks.World, Board: Earth1, StartedAt: contractOpens}}, s.utcAll(covering),
		"an era that starts at the window's end plays none of it")

	covering, err = s.store.Covering(ctx, clicks.World, Epoch, contractOpens.Add(-time.Second))
	s.Require().NoError(err)
	s.Equal([]Era{{Map: clicks.World, Board: Earth1, StartedAt: Epoch}}, s.utcAll(covering))
}

func (s *StoreContractSuite) TestAnArenaThatOpenedInTheWindowCoversItFromItsOpen() {
	ctx := s.T().Context()
	s.Require().NoError(s.store.Start(ctx, clicks.MapID("battle").On(Earth1.ID), contractOpens))

	covering, err := s.store.Covering(ctx, "battle", contractOpens.Add(-time.Hour), contractOpens.Add(time.Hour))
	s.Require().NoError(err)
	s.Equal([]Era{{Map: "battle", Board: Earth1, StartedAt: contractOpens}}, s.utcAll(covering))

	covering, err = s.store.Covering(ctx, "battle", contractOpens.Add(-2*time.Hour), contractOpens.Add(-time.Hour))
	s.Require().NoError(err)
	s.Empty(covering)
}

func (s *StoreContractSuite) utc(era Era) Era {
	era.StartedAt = era.StartedAt.UTC()
	return era
}

func (s *StoreContractSuite) utcAll(eras []Era) []Era {
	for i := range eras {
		eras[i] = s.utc(eras[i])
	}
	return eras
}
