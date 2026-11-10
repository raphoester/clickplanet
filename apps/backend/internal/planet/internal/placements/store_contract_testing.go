//go:build testing

package placements

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

const (
	contractAlice bonuses.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"
	contractBob   bonuses.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e60"
)

var contractTakenAt = time.Date(2026, 11, 10, 21, 0, 3, 123456000, time.UTC)

func (s *StoreContractSuite) placement(id byte, holder bonuses.Holder, after time.Duration) Placement {
	placement, err := New(clicks.PlacementID{15: id}, holder, contractTakenAt.Add(after))
	s.Require().NoError(err)
	return placement
}

func (s *StoreContractSuite) found(ids ...clicks.PlacementID) []Snapshot {
	found, err := s.store.Find(s.T().Context(), ids...)
	s.Require().NoError(err)
	snapshots := make([]Snapshot, 0, len(found))
	for _, placement := range found {
		snapshots = append(snapshots, placement.TakeSnapshot())
	}
	return snapshots
}

func (s *StoreContractSuite) all() []clicks.PlacementID {
	all, err := s.store.All(s.T().Context())
	s.Require().NoError(err)
	return all
}

func (s *StoreContractSuite) TestAPlacementUnderWayIsFoundAsItWasTaken() {
	taken := s.placement(1, contractAlice, 0)
	s.Require().NoError(s.store.Save(s.T().Context(), taken))

	s.Equal([]Snapshot{taken.TakeSnapshot()}, s.found(clicks.PlacementID{15: 1}))
	s.Equal([]clicks.PlacementID{{15: 1}}, s.all())
}

func (s *StoreContractSuite) TestAPlacementThatEndedIsNoRow() {
	taken := s.placement(1, contractAlice, 0)
	s.Require().NoError(s.store.Save(s.T().Context(), taken))

	s.Require().NoError(s.store.Save(s.T().Context(), taken.Ended()))

	s.Empty(s.found(clicks.PlacementID{15: 1}))
	s.Empty(s.all())
}

func (s *StoreContractSuite) TestFindAnswersOnlyThePlacementsAskedThatAreUnderWay() {
	s.Require().NoError(s.store.Save(s.T().Context(),
		s.placement(1, contractAlice, 0), s.placement(2, contractBob, time.Second), s.placement(3, contractAlice, 2*time.Second)))

	s.Equal([]Snapshot{s.placement(1, contractAlice, 0).TakeSnapshot(), s.placement(3, contractAlice, 2*time.Second).TakeSnapshot()},
		s.found(clicks.PlacementID{15: 3}, clicks.PlacementID{15: 1}, clicks.PlacementID{15: 9}))
	s.Empty(s.found())
}

func (s *StoreContractSuite) TestAllIsEveryPlacementUnderWayOldestFirst() {
	s.Require().NoError(s.store.Save(s.T().Context(),
		s.placement(1, contractAlice, 2*time.Second), s.placement(2, contractBob, 0), s.placement(3, contractAlice, time.Second)))
	s.Require().NoError(s.store.Save(s.T().Context(), s.placement(3, contractAlice, time.Second).Ended()))

	s.Equal([]clicks.PlacementID{{15: 2}, {15: 1}}, s.all())
}

func (s *StoreContractSuite) TestEndingAPlacementNeverKeptChangesNothing() {
	s.Require().NoError(s.store.Save(s.T().Context(), s.placement(1, contractAlice, 0).Ended()))

	s.Empty(s.all())
}
