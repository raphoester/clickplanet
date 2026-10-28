//go:build testing

package flagmissions

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
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
	contractCalendar = missions.NewCalendar(missions.Config{Cutoff: 21 * time.Hour})
	contractAt       = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	contractDay      = contractCalendar.DayOf(contractAt)
)

func (s *StoreContractSuite) keep(day missions.Day, flag string, drawn Day) DaySnapshot {
	kept, err := s.store.KeepDay(s.T().Context(), day, flag, drawn)
	s.Require().NoError(err)
	return kept.Snapshot()
}

func (s *StoreContractSuite) day(day missions.Day, flag string) DaySnapshot {
	read, err := s.store.Day(s.T().Context(), day, flag)
	s.Require().NoError(err)
	return read.Snapshot()
}

func (s *StoreContractSuite) drawn(id byte, order Order) Day {
	return Day{mission: &MissionSnapshot{ID: MissionID{15: id}, Order: order, Points: 2}}
}

func (s *StoreContractSuite) TestNoDayWasDrawnForADayAndFlagNeverKept() {
	_, err := s.store.Day(s.T().Context(), contractDay, "fr")

	s.ErrorIs(err, ErrNoDay)
}

func (s *StoreContractSuite) TestADayKeepsItsMissionOfEachKind() {
	for n, order := range []Order{TakeIn{Ground: "br", Goal: 300}, BreakShields{Flag: "dz", Goal: 150}} {
		flag := []string{"fr", "de"}[n]
		drawn := s.drawn(byte(n+1), order)

		s.Equal(drawn.Snapshot(), s.keep(contractDay, flag, drawn), flag)
		s.Equal(drawn.Snapshot(), s.day(contractDay, flag), flag)
	}
}

func (s *StoreContractSuite) TestADayWithNoMissionIsKept() {
	s.Equal(DaySnapshot{}, s.keep(contractDay, "fr", Day{}))
	s.Equal(DaySnapshot{}, s.day(contractDay, "fr"))
}

func (s *StoreContractSuite) TestTheFirstDayKeptForADayAndFlagWins() {
	first := s.drawn(1, TakeIn{Ground: "br", Goal: 300})
	s.keep(contractDay, "fr", first)

	s.Equal(first.Snapshot(), s.keep(contractDay, "fr", s.drawn(2, BreakShields{Flag: "dz", Goal: 150})))
}

func (s *StoreContractSuite) TestDaysAreKeptByDayAndFlag() {
	s.keep(contractDay, "fr", s.drawn(1, TakeIn{Ground: "br", Goal: 300}))

	_, err := s.store.Day(s.T().Context(), contractDay, "de")
	s.ErrorIs(err, ErrNoDay)
	_, err = s.store.Day(s.T().Context(), contractDay.Previous(), "fr")
	s.ErrorIs(err, ErrNoDay)
}

func (s *StoreContractSuite) TestAChangeIsKept() {
	s.keep(contractDay, "fr", s.drawn(1, TakeIn{Ground: "br", Goal: 300}))
	changed := DaySnapshot{
		Mission:  &MissionSnapshot{ID: MissionID{15: 1}, Order: TakeIn{Ground: "br", Goal: 300}, Points: 2},
		Progress: 300, Done: true, Takes: 1_200, TakesBelow: 99, Strikes: 40, StrikesUntil: contractAt,
	}

	s.Require().NoError(s.store.ChangeDay(s.T().Context(), contractDay, "fr", func(Day) Day {
		day, err := DayOf(changed)
		s.Require().NoError(err)
		return day
	}))

	s.Equal(changed, s.day(contractDay, "fr"))
}

func (s *StoreContractSuite) TestAChangeOfADayWithNoMissionKeepsItsCounts() {
	s.keep(contractDay, "fr", Day{})

	s.Require().NoError(s.store.ChangeDay(s.T().Context(), contractDay, "fr", func(Day) Day {
		return Day{takes: 7, takesBelow: 8}
	}))

	s.Equal(DaySnapshot{Takes: 7, TakesBelow: 8}, s.day(contractDay, "fr"))
}

func (s *StoreContractSuite) TestADayNeverKeptCannotChange() {
	err := s.store.ChangeDay(s.T().Context(), contractDay, "fr", func(day Day) Day { return day })

	s.ErrorIs(err, ErrNoDay)
}
