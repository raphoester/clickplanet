//go:build testing

package missions

import (
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
		Territories:  []Territory{{Flag: "dz", Tiles: 9_000}},
		CountrySizes: []Holding{{Country: "br", Tiles: 9_000}, {Country: "is", Tiles: 1_200}},
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

func (s *StoreContractSuite) frMission(n int) MissionID {
	return Draw(contractDay, "fr", contractWorld).Snapshot()[n].ID
}

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

func (s *StoreContractSuite) TestABriefKeepsAsManyMissionsAsItHas() {
	orders := []Order{
		TakeFor{Flag: "fr", Goal: 10},
		TakeFrom{Flag: "dz", Goal: 11},
		TakeIn{Ground: "br", Goal: 12},
		TakeFor{Flag: "fr", Goal: 13},
		TakeIn{Ground: "is", Goal: 14},
	}
	snapshots := make([]MissionSnapshot, 0, len(orders))
	for n, order := range orders {
		snapshots = append(snapshots, MissionSnapshot{
			ID: drawnID(contractDay, "fr", n), Order: order, Reward: Reward{Kind: RewardShields, Amount: uint32(n + 1)},
		})
	}
	brief, err := BriefOf(snapshots)
	s.Require().NoError(err)

	s.Equal(snapshots, s.keep(contractDay, "fr", brief).Snapshot(), "each kind with its own fields, in the order drawn")
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
	s.Equal(SheetSnapshot{Progress: map[MissionID]uint32{}, Paid: cpcolls.NewSet[MissionID]()}, s.sheet(contractDay, 1))
}

func (s *StoreContractSuite) TestChangesToASheetAddUp() {
	s.takeFor(contractDay, 1, Take{Flag: "fr", Previous: "dz", Ground: "es", At: contractAt})
	s.takeFor(contractDay, 1, Take{Flag: "fr", Ground: "es", At: contractAt})

	s.Equal(SheetSnapshot{
		Flag: "fr", Progress: map[MissionID]uint32{s.frMission(0): 2, s.frMission(1): 1}, Paid: cpcolls.NewSet[MissionID](),
		TakesBelow: 3,
	}, s.sheet(contractDay, 1))
}

func (s *StoreContractSuite) TestTheMarkOfTheLastTakeCountedIsKept() {
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt, Position: 41})
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt, Position: 41})

	s.Equal(uint64(42), s.sheet(contractDay, 1).TakesBelow)
	s.Equal(uint32(1), s.sheet(contractDay, 1).Progress[s.frMission(0)], "the second telling counted nothing")
}

func (s *StoreContractSuite) TestASheetIsKeptByDayAndAccount() {
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt})

	s.False(SheetOf(s.sheet(contractDay, 2)).Enlisted())
	s.False(SheetOf(s.sheet(contractNextDay, 1)).Enlisted())
}

func (s *StoreContractSuite) pay(day Day, account byte, mission MissionID) {
	s.Require().NoError(s.store.ChangeSheet(s.T().Context(), day, s.account(account), func(sheet Sheet) Sheet {
		snapshot := sheet.Snapshot()
		snapshot.Flag = "fr"
		snapshot.Paid.Add(mission)
		return SheetOf(snapshot)
	}))
}

func (s *StoreContractSuite) TestAPaymentIsKept() {
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt})

	s.pay(contractDay, 1, s.frMission(0))
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt})

	s.Equal(cpcolls.NewSet(s.frMission(0)), s.sheet(contractDay, 1).Paid)
	s.Equal(uint32(2), s.sheet(contractDay, 1).Progress[s.frMission(0)], "and the takes after it")
}

func (s *StoreContractSuite) TestDeletingAnAccountForgetsItsDays() {
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt})
	s.takeFor(contractDay, 2, Take{Flag: "fr", At: contractAt})
	s.pay(contractDay, 1, s.frMission(0))

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), s.account(1)))

	s.False(SheetOf(s.sheet(contractDay, 1)).Enlisted())
	s.True(s.sheet(contractDay, 1).Paid.Empty())
	s.True(SheetOf(s.sheet(contractDay, 2)).Enlisted(), "another account keeps its day")
}
