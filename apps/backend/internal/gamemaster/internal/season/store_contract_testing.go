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

func kept(number Number, brk, length time.Duration) Season {
	return Snapshot{Number: number, Break: brk, Length: length}.Restore()
}

func (s *StoreContractSuite) save(seasons ...Season) {
	for _, season := range seasons {
		s.Require().NoError(s.store.Save(s.T().Context(), season))
	}
}

func (s *StoreContractSuite) read() []Snapshot {
	calendar, err := s.store.Calendar(s.T().Context())
	s.Require().NoError(err)
	return calendar.TakeSnapshot()
}

func (s *StoreContractSuite) TestNoSeasonIsAnEmptyCalendar() {
	s.Empty(s.read())
}

func (s *StoreContractSuite) TestTheSeasonsAreReadAsTheyWereKeptByNumber() {
	first := kept(0, 0, 28*24*time.Hour+23*time.Hour+123456*time.Microsecond)
	second := kept(1, 3*time.Hour, 7*24*time.Hour)
	s.save(second, first)

	s.Equal([]Snapshot{first.TakeSnapshot(), second.TakeSnapshot()}, s.read())
}

func (s *StoreContractSuite) TestASeasonKeptAgainReplacesItsPlan() {
	s.save(kept(0, 0, time.Hour))
	moved := kept(0, time.Minute, 5*time.Hour)
	s.save(moved)

	s.Equal([]Snapshot{moved.TakeSnapshot()}, s.read())
}

func (s *StoreContractSuite) TestADeletedSeasonIsGone() {
	first, second := kept(0, 0, time.Hour), kept(1, 0, time.Hour)
	s.save(first, second)

	s.Require().NoError(s.store.Delete(s.T().Context(), 1))

	s.Equal([]Snapshot{first.TakeSnapshot()}, s.read())
}

func (s *StoreContractSuite) TestAFinaleIsKeptWithItsSeason() {
	finale := Finale{Length: 2 * time.Hour, Set: "rush", Description: "Triple points.", Times: 3}
	planned := Snapshot{Number: 0, Length: 24 * time.Hour, Finale: finale}.Restore()
	s.save(planned, kept(1, time.Hour, time.Hour))

	s.Equal([]Snapshot{planned.TakeSnapshot(), kept(1, time.Hour, time.Hour).TakeSnapshot()}, s.read())
}

func (s *StoreContractSuite) TestAFinaleKeptAgainIsReplacedOrGone() {
	s.save(Snapshot{Number: 0, Length: 24 * time.Hour, Finale: Finale{Length: time.Hour, Set: "rush", Times: 3}}.Restore())
	changed := Snapshot{Number: 0, Length: 24 * time.Hour, Finale: Finale{Length: 3 * time.Hour, Set: "storm", Description: "Double points.", Times: 2}}.Restore()
	s.save(changed)
	s.Equal([]Snapshot{changed.TakeSnapshot()}, s.read())

	s.save(kept(0, 0, 24*time.Hour))
	s.Equal([]Snapshot{kept(0, 0, 24*time.Hour).TakeSnapshot()}, s.read())
}

func (s *StoreContractSuite) TestADeletedSeasonTakesItsFinale() {
	s.save(kept(0, 0, time.Hour), Snapshot{Number: 1, Length: time.Hour, Finale: Finale{Length: time.Hour, Set: "rush", Times: 3}}.Restore())

	s.Require().NoError(s.store.Delete(s.T().Context(), 1))
	s.save(kept(1, 0, time.Hour))

	s.Equal([]Snapshot{kept(0, 0, time.Hour).TakeSnapshot(), kept(1, 0, time.Hour).TakeSnapshot()}, s.read())
}
