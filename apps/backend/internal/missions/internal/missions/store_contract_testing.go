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

func (s *StoreContractSuite) keep(day Day, account byte, brief Brief) Brief {
	kept, err := s.store.KeepBrief(s.T().Context(), day, s.account(account), brief)
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
	brief := Draw(day, s.account(account), take.Flag, contractWorld)
	s.Require().NoError(s.store.ChangeSheet(s.T().Context(), day, s.account(account),
		func(sheet Sheet) Sheet { return sheet.WithTake(take, brief) }))
}

func (s *StoreContractSuite) frMission(account byte, n int) MissionID {
	return Draw(contractDay, s.account(account), "fr", contractWorld).Snapshot()[n].ID
}

func (s *StoreContractSuite) noBrief(day Day, account byte) {
	_, err := s.store.Brief(s.T().Context(), day, s.account(account))
	s.ErrorIs(err, ErrNoBrief)
}

func (s *StoreContractSuite) TestNoBriefWasDrawnForADayAndAccountNeverKept() {
	s.noBrief(contractDay, 1)
}

func (s *StoreContractSuite) TestAKeptBriefReadsBack() {
	drawn := Draw(contractDay, s.account(1), "fr", contractWorld)

	s.Equal(drawn.Snapshot(), s.keep(contractDay, 1, drawn).Snapshot())

	read, err := s.store.Brief(s.T().Context(), contractDay, s.account(1))
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
			ID: drawnID(contractDay, s.account(1), n), Order: order, Reward: Reward{Kind: RewardShields, Amount: uint32(n + 1)},
		})
	}
	brief, err := BriefOf(snapshots)
	s.Require().NoError(err)

	s.Equal(snapshots, s.keep(contractDay, 1, brief).Snapshot(), "each kind with its own fields, in the order drawn")
}

func (s *StoreContractSuite) TestTheFirstBriefKeptForADayAndAccountWins() {
	first := Draw(contractDay, s.account(1), "fr", contractWorld)
	s.keep(contractDay, 1, first)

	again := s.keep(contractDay, 1, Draw(contractDay, s.account(1), "de", World{}))

	s.Equal(first.Snapshot(), again.Snapshot())
}

func (s *StoreContractSuite) TestBriefsAreKeptByDayAndAccount() {
	s.keep(contractDay, 1, Draw(contractDay, s.account(1), "fr", contractWorld))

	s.noBrief(contractDay, 2)
	s.noBrief(contractNextDay, 1)
}

func (s *StoreContractSuite) TestASheetNeverChangedIsEmpty() {
	s.Equal(SheetSnapshot{Progress: map[MissionID]uint32{}, Paid: cpcolls.NewSet[MissionID]()}, s.sheet(contractDay, 1))
}

func (s *StoreContractSuite) TestChangesToASheetAddUp() {
	s.takeFor(contractDay, 1, Take{Flag: "fr", Previous: "dz", Ground: "es", At: contractAt})
	s.takeFor(contractDay, 1, Take{Flag: "fr", Ground: "es", At: contractAt})

	s.Equal(SheetSnapshot{
		Progress: map[MissionID]uint32{s.frMission(1, 0): 2, s.frMission(1, 1): 1}, Paid: cpcolls.NewSet[MissionID](),
		TakesBelow: 3,
	}, s.sheet(contractDay, 1))
}

func (s *StoreContractSuite) TestTheMarkOfTheLastTakeCountedIsKept() {
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt, Position: 41})
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt, Position: 41})

	s.Equal(uint64(42), s.sheet(contractDay, 1).TakesBelow)
	s.Equal(uint32(1), s.sheet(contractDay, 1).Progress[s.frMission(1, 0)], "the second telling counted nothing")
}

func (s *StoreContractSuite) TestASheetIsKeptByDayAndAccount() {
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt})

	s.Empty(s.sheet(contractDay, 2).Progress)
	s.Empty(s.sheet(contractNextDay, 1).Progress)
}

func (s *StoreContractSuite) pay(day Day, account byte, mission MissionID) {
	s.Require().NoError(s.store.ChangeSheet(s.T().Context(), day, s.account(account), func(sheet Sheet) Sheet {
		snapshot := sheet.Snapshot()
		snapshot.Paid.Add(mission)
		return SheetOf(snapshot)
	}))
}

func (s *StoreContractSuite) TestAPaymentIsKept() {
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt})

	s.pay(contractDay, 1, s.frMission(1, 0))
	s.takeFor(contractDay, 1, Take{Flag: "fr", At: contractAt})

	s.Equal(cpcolls.NewSet(s.frMission(1, 0)), s.sheet(contractDay, 1).Paid)
	s.Equal(uint32(2), s.sheet(contractDay, 1).Progress[s.frMission(1, 0)], "and the takes after it")
}

func (s *StoreContractSuite) paidInAll(account byte) uint64 {
	paid, err := s.store.PaidInAll(s.T().Context(), s.account(account))
	s.Require().NoError(err)
	return paid
}

func (s *StoreContractSuite) TestTheMissionsPaidAddUpOverEveryDay() {
	s.Zero(s.paidInAll(1))

	for _, day := range []Day{contractDay, contractNextDay} {
		s.takeFor(day, 1, Take{Flag: "fr", At: contractAt})
	}
	s.takeFor(contractDay, 2, Take{Flag: "fr", At: contractAt})
	s.pay(contractDay, 1, s.frMission(1, 0))
	s.pay(contractDay, 1, s.frMission(1, 1))
	s.pay(contractNextDay, 1, Draw(contractNextDay, s.account(1), "fr", contractWorld).Snapshot()[0].ID)
	s.pay(contractDay, 2, s.frMission(2, 0))

	s.Equal(uint64(3), s.paidInAll(1))
	s.Equal(uint64(1), s.paidInAll(2))
}

func (s *StoreContractSuite) TestDeletingAnAccountForgetsItsDays() {
	for account := range byte(2) {
		s.keep(contractDay, account+1, Draw(contractDay, s.account(account+1), "fr", contractWorld))
		s.takeFor(contractDay, account+1, Take{Flag: "fr", At: contractAt})
	}
	s.pay(contractDay, 1, s.frMission(1, 0))

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), s.account(1)))

	s.noBrief(contractDay, 1)
	s.Empty(s.sheet(contractDay, 1).Progress)
	s.True(s.sheet(contractDay, 1).Paid.Empty())
	s.Zero(s.paidInAll(1))
	_, err := s.store.Brief(s.T().Context(), contractDay, s.account(2))
	s.Require().NoError(err, "another account keeps its day")
	s.NotEmpty(s.sheet(contractDay, 2).Progress)
}
