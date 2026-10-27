//go:build testing

package season

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

var contractStart = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

func kept(number Number, startsAt, endsAt time.Time) Season {
	return Snapshot{Number: number, StartsAt: startsAt, EndsAt: endsAt}.Restore()
}

func (s *StoreContractSuite) save(seasons ...Season) {
	for _, season := range seasons {
		s.Require().NoError(s.store.Save(s.T().Context(), season))
	}
}

func (s *StoreContractSuite) read() []Snapshot {
	calendar, err := s.store.Calendar(s.T().Context())
	s.Require().NoError(err)
	snapshots := calendar.TakeSnapshot()
	for i := range snapshots {
		snapshots[i].StartsAt, snapshots[i].EndsAt = snapshots[i].StartsAt.UTC(), snapshots[i].EndsAt.UTC()
	}
	return snapshots
}

func (s *StoreContractSuite) TestNoSeasonIsAnEmptyCalendar() {
	s.Empty(s.read())
}

func (s *StoreContractSuite) TestTheSeasonsAreReadAsTheyWereKeptByNumber() {
	first := kept(0, contractStart, contractStart.Add(time.Hour+123456*time.Microsecond))
	second := kept(1, contractStart.Add(2*time.Hour), contractStart.Add(3*time.Hour))
	s.save(second, first)

	s.Equal([]Snapshot{first.TakeSnapshot(), second.TakeSnapshot()}, s.read())
}

func (s *StoreContractSuite) TestASeasonKeptAgainReplacesItsWindow() {
	s.save(kept(0, contractStart, contractStart.Add(time.Hour)))
	moved := kept(0, contractStart, contractStart.Add(5*time.Hour))
	s.save(moved)

	s.Equal([]Snapshot{moved.TakeSnapshot()}, s.read())
}

func (s *StoreContractSuite) TestADeletedSeasonIsGone() {
	first, second := kept(0, contractStart, contractStart.Add(time.Hour)), kept(1, contractStart.Add(time.Hour), contractStart.Add(2*time.Hour))
	s.save(first, second)

	s.Require().NoError(s.store.Delete(s.T().Context(), 1))

	s.Equal([]Snapshot{first.TakeSnapshot()}, s.read())
}
