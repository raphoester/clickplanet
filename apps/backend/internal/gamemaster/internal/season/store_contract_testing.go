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
