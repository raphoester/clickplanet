//go:build testing

package days

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

var contractAt = time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)

func (s *StoreContractSuite) TestADayIsFoundFromItsStartToBeforeItsEnd() {
	day := TestingDayAt(contractAt)
	s.Require().NoError(s.store.KeepDay(s.T().Context(), day))

	for _, at := range []time.Time{day.Snapshot().StartsAt, contractAt, day.Snapshot().EndsAt.Add(-time.Microsecond)} {
		found, err := s.store.DayAt(s.T().Context(), at)
		s.Require().NoError(err)
		s.Equal(day, found, "%s", at)
	}
}

func (s *StoreContractSuite) TestNoDayRunsBeforeOrAfterTheDaysTold() {
	day := TestingDayAt(contractAt)
	s.Require().NoError(s.store.KeepDay(s.T().Context(), day))

	for _, at := range []time.Time{day.Snapshot().StartsAt.Add(-time.Microsecond), day.Snapshot().EndsAt} {
		_, err := s.store.DayAt(s.T().Context(), at)
		s.Require().ErrorIs(err, ErrNoDay, "%s", at)
	}
}

func (s *StoreContractSuite) TestEachDayIsFoundAtItsOwnTimes() {
	today, tomorrow := TestingDayAt(contractAt), TestingDayAt(contractAt.Add(24*time.Hour))
	s.Require().NoError(s.store.KeepDay(s.T().Context(), tomorrow))
	s.Require().NoError(s.store.KeepDay(s.T().Context(), today))

	found, err := s.store.DayAt(s.T().Context(), today.Snapshot().EndsAt)
	s.Require().NoError(err)
	s.Equal(tomorrow, found)
}

func (s *StoreContractSuite) TestADayToldAgainKeepsItsNewEnd() {
	day := TestingDayAt(contractAt)
	cut := day.Snapshot()
	cut.EndsAt = contractAt.Add(time.Hour)
	moved, err := DayOf(cut)
	s.Require().NoError(err)

	s.Require().NoError(s.store.KeepDay(s.T().Context(), day))
	s.Require().NoError(s.store.KeepDay(s.T().Context(), moved))

	found, err := s.store.DayAt(s.T().Context(), contractAt)
	s.Require().NoError(err)
	s.Equal(moved, found)
	_, err = s.store.DayAt(s.T().Context(), contractAt.Add(2*time.Hour))
	s.Require().ErrorIs(err, ErrNoDay, "the day was cut")
}

func (s *StoreContractSuite) TestTheDayBeforeIsTheOneThatEndedAsTheDayStarted() {
	yesterday, today, tomorrow := TestingDayAt(contractAt.Add(-24*time.Hour)), TestingDayAt(contractAt), TestingDayAt(contractAt.Add(24*time.Hour))
	for _, day := range []Day{tomorrow, yesterday, today} {
		s.Require().NoError(s.store.KeepDay(s.T().Context(), day))
	}

	found, err := s.store.DayBefore(s.T().Context(), today)
	s.Require().NoError(err)
	s.Equal(yesterday, found)
	_, err = s.store.DayBefore(s.T().Context(), yesterday)
	s.Require().ErrorIs(err, ErrNoDay, "no day was told before it")
}
