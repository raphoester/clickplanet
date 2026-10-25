//go:build testing

package missions

import (
	"context"
	"errors"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store    Store
	position uint64
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
	s.position = 0
}

var (
	contractCalendar = NewCalendar(Config{Cutoff: 21 * time.Hour})
	contractAt       = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	contractDay      = contractCalendar.DayOf(contractAt)
	contractNextDay  = contractCalendar.DayOf(contractAt.AddDate(0, 0, 1))
	contractWorld    = World{
		Territories: []Holding{{Country: "dz", Tiles: 9_000}},
		Grounds:     []Holding{{Country: "br", Tiles: 9_000}, {Country: "is", Tiles: 1_200}},
	}
)

func (s *StoreContractSuite) account(n byte) AccountID {
	return AccountID{15: n}
}

func (s *StoreContractSuite) keep(day Day, flag string, brief Brief) Brief {
	kept, err := s.store.KeepBrief(s.T().Context(), day, flag, brief)
	s.Require().NoError(err)
	return kept
}

func (s *StoreContractSuite) sheet(day Day, account byte) SheetSnapshot {
	sheet, err := s.store.Sheet(s.T().Context(), day, s.account(account))
	s.Require().NoError(err)
	return sheet.Snapshot()
}

func (s *StoreContractSuite) takeFor(day Day, account byte, take Take) {
	if take.Position == 0 {
		s.position++
		take.Position = s.position
	}
	brief := Draw(day, take.Flag, contractWorld)
	s.Require().NoError(s.store.ChangeSheet(s.T().Context(), day, s.account(account),
		func(sheet Sheet) Sheet { return sheet.WithTake(take, brief) }))
}

func paid(context.Context) error { return nil }

func (s *StoreContractSuite) TestNoBriefWasDrawnForADayAndFlagNeverKept() {
	_, err := s.store.Brief(s.T().Context(), contractDay, "fr")

	s.ErrorIs(err, ErrNoBrief)
}

func (s *StoreContractSuite) TestAKeptBriefReadsBack() {
	drawn := Draw(contractDay, "fr", contractWorld)

	s.Equal(drawn.Snapshot(), s.keep(contractDay, "fr", drawn).Snapshot())

	read, err := s.store.Brief(s.T().Context(), contractDay, "fr")
	s.Require().NoError(err)
	s.Equal(drawn.Snapshot(), read.Snapshot())
}

func (s *StoreContractSuite) TestTheFirstBriefKeptForADayAndFlagWins() {
	first := Draw(contractDay, "fr", contractWorld)
	s.keep(contractDay, "fr", first)

	again := s.keep(contractDay, "fr", Draw(contractDay, "fr", World{}))

	s.Equal(first.Snapshot(), again.Snapshot())
}

func (s *StoreContractSuite) TestBriefsAreKeptByDayAndFlag() {
	s.keep(contractDay, "fr", Draw(contractDay, "fr", contractWorld))

	_, err := s.store.Brief(s.T().Context(), contractDay, "de")
	s.ErrorIs(err, ErrNoBrief)
	_, err = s.store.Brief(s.T().Context(), contractNextDay, "fr")
	s.ErrorIs(err, ErrNoBrief)
}

func (s *StoreContractSuite) TestASheetNeverChangedIsEmpty() {
	s.Equal(SheetSnapshot{Taken: map[Slot]uint32{}, Collected: cpcolls.NewSet[Slot]()}, s.sheet(contractDay, 1))
}

func (s *StoreContractSuite) TestChangesToASheetAddUp() {
	s.takeFor(contractDay, 1, Take{Flag: "fr", Previous: "dz", Ground: "es", At: contractAt})
	s.takeFor(contractDay, 1, Take{Flag: "fr", Ground: "es", At: contractAt})

	s.Equal(SheetSnapshot{
		Flag: "fr", Taken: map[Slot]uint32{1: 2, 2: 1}, Collected: cpcolls.NewSet[Slot](), TakesBelow: 3,
	}, s.sheet(contractDay, 1))
}

func (s *StoreContractSuite) TestTheMarkOfTheLastTakeCountedIsKept() {
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt, Position: 41})
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt, Position: 41})

	s.Equal(uint64(42), s.sheet(contractDay, 1).TakesBelow)
	s.Equal(uint32(1), s.sheet(contractDay, 1).Taken[1], "the second telling counted nothing")
}

func (s *StoreContractSuite) TestASheetIsKeptByDayAndAccount() {
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt})

	s.False(SheetOf(s.sheet(contractDay, 2)).Enlisted())
	s.False(SheetOf(s.sheet(contractNextDay, 1)).Enlisted())
}

func (s *StoreContractSuite) TestACollectionIsKeptOnceItIsPaid() {
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt})

	s.Require().NoError(s.store.Collect(s.T().Context(), contractDay, s.account(1), 1, contractAt, paid))

	s.Equal(cpcolls.NewSet[Slot](1), s.sheet(contractDay, 1).Collected)
}

func (s *StoreContractSuite) TestACollectionThatCouldNotBePaidIsNotKept() {
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt})
	unpaid := errors.New("the planet is down")

	err := s.store.Collect(s.T().Context(), contractDay, s.account(1), 1, contractAt,
		func(context.Context) error { return unpaid })

	s.ErrorIs(err, unpaid)
	s.True(s.sheet(contractDay, 1).Collected.Empty())
}

func (s *StoreContractSuite) TestASlotIsCollectedOncePaidOnce() {
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt})
	s.Require().NoError(s.store.Collect(s.T().Context(), contractDay, s.account(1), 1, contractAt, paid))

	payments := 0
	err := s.store.Collect(s.T().Context(), contractDay, s.account(1), 1, contractAt,
		func(context.Context) error { payments++; return nil })

	s.ErrorIs(err, ErrCollected)
	s.Zero(payments)
}

func (s *StoreContractSuite) TestDeletingAnAccountForgetsItsDays() {
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt})
	s.takeFor(contractDay, 2, Take{Flag: "fr", At: contractAt})
	s.Require().NoError(s.store.Collect(s.T().Context(), contractDay, s.account(1), 1, contractAt, paid))

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), s.account(1)))

	s.False(SheetOf(s.sheet(contractDay, 1)).Enlisted())
	s.True(s.sheet(contractDay, 1).Collected.Empty())
	s.True(SheetOf(s.sheet(contractDay, 2)).Enlisted(), "another account keeps its day")
}
