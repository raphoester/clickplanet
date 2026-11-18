//go:build testing

package missions

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
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
	contractAt      = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	contractDay     = days.TestingDayAt(contractAt)
	contractNextDay = days.TestingDayAt(contractAt.AddDate(0, 0, 1))
	contractWorld   = World{
		Territories:  []Territory{{Flag: "dz", Tiles: 9_000}},
		CountrySizes: []Holding{{Country: "br", Tiles: 9_000}, {Country: "is", Tiles: 1_200}},
	}
)

func (s *StoreContractSuite) account(n byte) AccountID {
	return AccountID{15: n}
}

func (s *StoreContractSuite) keep(day days.Day, account byte, brief Brief) Brief {
	kept, err := s.store.KeepBrief(s.T().Context(), day, s.account(account), brief)
	s.Require().NoError(err)
	return kept
}

func (s *StoreContractSuite) sheet(day days.Day, account byte) SheetSnapshot {
	sheet, err := s.store.Sheet(s.T().Context(), day, s.account(account))
	s.Require().NoError(err)
	return sheet.Snapshot()
}

func (s *StoreContractSuite) takeFor(day days.Day, account byte, take Take) {
	if take.Position == 0 {
		s.position++
		take.Position = s.position
	}
	brief := Draw(day, s.account(account), take.Flag, contractWorld)
	s.Require().NoError(s.store.ChangeSheet(s.T().Context(), day, s.account(account),
		func(sheet Sheet) Sheet { return sheet.WithFact(take, brief) }))
}

func (s *StoreContractSuite) frMission(account byte, n int) MissionID {
	return Draw(contractDay, s.account(account), "fr", contractWorld).Snapshot()[n].ID
}

func (s *StoreContractSuite) noBrief(day days.Day, account byte) {
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
		TakeFromIn{Flag: "dz", Ground: "br", Goal: 15},
		CatchBoxes{Goal: 16},
		AnswerQuizzes{Goal: 17},
		BombIn{Ground: "br", Goal: 18},
		ShieldTiles{Shields: 10, Goal: 19},
		FortifyTiles{Goal: 20},
	}
	snapshots := make([]MissionSnapshot, 0, len(orders))
	for n, order := range orders {
		snapshots = append(snapshots, MissionSnapshot{ID: drawnID(contractDay, s.account(1), n), Order: order})
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
	s.takeFor(contractDay, 1, Take{Flag: "fr", Previous: "dz", Ground: "es"})
	s.takeFor(contractDay, 1, Take{Flag: "fr", Ground: "es"})

	s.Equal(SheetSnapshot{
		Progress: map[MissionID]uint32{s.frMission(1, 0): 2, s.frMission(1, 1): 1}, Paid: cpcolls.NewSet[MissionID](),
		Counted: Counted{TakesBelow: LedgerMark{Position: 2, Index: 1}},
	}, s.sheet(contractDay, 1))
}

func (s *StoreContractSuite) TestEveryMarkOfWhatWasCountedIsKept() {
	counted := Counted{
		TakesBelow: LedgerMark{Position: 5, Index: 2}, BombsBelow: 6, ShieldsBelow: 7, FortifiesBelow: 8,
		BoxesUntil: contractAt, AnswersUntil: contractAt.Add(time.Microsecond),
	}

	s.Require().NoError(s.store.ChangeSheet(s.T().Context(), contractDay, s.account(1),
		func(Sheet) Sheet { return SheetOf(SheetSnapshot{Counted: counted}) }))

	s.Equal(counted, s.sheet(contractDay, 1).Counted)
}

func (s *StoreContractSuite) TestASheetIsKeptByDayAndAccount() {
	s.takeFor(contractDay, 1, Take{Flag: "fr"})

	s.Empty(s.sheet(contractDay, 2).Progress)
	s.Empty(s.sheet(contractNextDay, 1).Progress)
}

func (s *StoreContractSuite) pay(day days.Day, account byte, mission MissionID) {
	s.Require().NoError(s.store.ChangeSheet(s.T().Context(), day, s.account(account), func(sheet Sheet) Sheet {
		snapshot := sheet.Snapshot()
		snapshot.Paid.Add(mission)
		return SheetOf(snapshot)
	}))
}

func (s *StoreContractSuite) TestAPaymentIsKept() {
	s.takeFor(contractDay, 1, Take{Flag: "fr"})

	s.pay(contractDay, 1, s.frMission(1, 0))
	s.takeFor(contractDay, 1, Take{Flag: "fr"})

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

	for _, day := range []days.Day{contractDay, contractNextDay} {
		s.takeFor(day, 1, Take{Flag: "fr"})
	}
	s.takeFor(contractDay, 2, Take{Flag: "fr"})
	s.pay(contractDay, 1, s.frMission(1, 0))
	s.pay(contractDay, 1, s.frMission(1, 1))
	s.pay(contractNextDay, 1, Draw(contractNextDay, s.account(1), "fr", contractWorld).Snapshot()[0].ID)
	s.pay(contractDay, 2, s.frMission(2, 0))

	s.Equal(uint64(3), s.paidInAll(1))
	s.Equal(uint64(1), s.paidInAll(2))
}

func (s *StoreContractSuite) done(account byte) Done {
	done, err := s.store.DoneBy(s.T().Context(), s.account(account))
	s.Require().NoError(err)
	return done
}

func (s *StoreContractSuite) doneAfter(after AccountID, limit int) []AccountDone {
	page, err := s.store.DoneAfter(s.T().Context(), after, limit)
	s.Require().NoError(err)
	return page
}

// One mission of every kind, each with a goal of its own, the first kind twice; each taken to progress of its own.
func (s *StoreContractSuite) keepEveryKind(day days.Day, account byte, progress ...uint32) {
	orders := []Order{
		TakeFor{Flag: "fr", Goal: 120},
		TakeFor{Flag: "fr", Goal: 200},
		TakeFrom{Flag: "dz", Goal: 80},
		TakeIn{Ground: "br", Goal: 300},
		TakeFromIn{Flag: "dz", Ground: "br", Goal: 40},
		CatchBoxes{Goal: 3},
		AnswerQuizzes{Goal: 2},
		BombIn{Ground: "br", Goal: 1},
		ShieldTiles{Shields: 10, Goal: 2},
		FortifyTiles{Goal: 20},
	}
	snapshots := make([]MissionSnapshot, 0, len(orders))
	reached := map[MissionID]uint32{}
	for n, order := range orders {
		id := drawnID(day, s.account(account), n)
		snapshots = append(snapshots, MissionSnapshot{ID: id, Order: order})
		if n < len(progress) && progress[n] > 0 {
			reached[id] = progress[n]
		}
	}
	brief, err := BriefOf(snapshots)
	s.Require().NoError(err)
	s.keep(day, account, brief)
	s.Require().NoError(s.store.ChangeSheet(s.T().Context(), day, s.account(account), func(Sheet) Sheet {
		// A mark, so postgres keeps the day's row the progress hangs from.
		return SheetOf(SheetSnapshot{Progress: reached, Counted: Counted{BombsBelow: 1}})
	}))
}

func (s *StoreContractSuite) TestAnAccountWithNoMissionDoneHasDoneNothing() {
	s.keepEveryKind(contractDay, 1, 119, 199, 79)

	s.Equal(NothingDone(), s.done(1), "each one short of its goal")
	s.Equal(NothingDone(), s.done(2), "an account never seen")
}

func (s *StoreContractSuite) TestTheMissionsDoneAreCountedByTheKindTheirBriefKeepsWithTheirGoals() {
	s.keepEveryKind(contractDay, 1, 120, 200, 80, 300, 40, 3, 2, 1, 2, 20)
	s.keepEveryKind(contractNextDay, 1, 120, 10, 0, 0, 0, 3)

	s.Equal(Done{InAll: 12, ByKind: map[Kind]KindDone{
		KindTakeFor:       {Done: 3, Goals: 440},
		KindTakeFrom:      {Done: 1, Goals: 80},
		KindTakeIn:        {Done: 1, Goals: 300},
		KindTakeFromIn:    {Done: 1, Goals: 40},
		KindCatchBoxes:    {Done: 2, Goals: 6},
		KindAnswerQuizzes: {Done: 1, Goals: 2},
		KindBombIn:        {Done: 1, Goals: 1},
		KindShieldTiles:   {Done: 1, Goals: 2},
		KindFortifyTiles:  {Done: 1, Goals: 20},
	}}, s.done(1), "every kind, over every day; a mission short of its goal left out")
	s.Len(s.done(1).ByKind, len(TestingKinds()))
}

func (s *StoreContractSuite) TestAMissionDoneCountsWhetherOrNotItWasPaid() {
	s.keepEveryKind(contractDay, 1, 120)
	unpaid := s.done(1)

	s.pay(contractDay, 1, drawnID(contractDay, s.account(1), 0))

	s.Equal(unpaid, s.done(1))
	s.Equal(uint64(1), unpaid.InAll)
}

func (s *StoreContractSuite) TestTheAccountsWithAMissionDoneArePagedInOrderWithTheirTotals() {
	for _, account := range []byte{3, 1, 2} {
		s.keepEveryKind(contractDay, account, 120, 200, 80)
	}
	s.keepEveryKind(contractDay, 4, 1, 1, 1)

	first := s.doneAfter(AccountID{}, 2)
	second := s.doneAfter(first[len(first)-1].Account, 2)

	s.Equal([]AccountDone{
		{Account: s.account(1), Done: s.done(1)},
		{Account: s.account(2), Done: s.done(2)},
	}, first)
	s.Equal([]AccountDone{{Account: s.account(3), Done: s.done(3)}}, second, "an account with no mission done is left out")
	s.Equal(uint64(3), first[0].Done.InAll)
	s.Equal(KindDone{Done: 2, Goals: 320}, first[0].Done.ByKind[KindTakeFor])
	s.Empty(s.doneAfter(s.account(3), 2))
}

func (s *StoreContractSuite) TestDeletingAnAccountForgetsItsDays() {
	for account := range byte(2) {
		s.keep(contractDay, account+1, Draw(contractDay, s.account(account+1), "fr", contractWorld))
		s.takeFor(contractDay, account+1, Take{Flag: "fr"})
	}
	s.pay(contractDay, 1, s.frMission(1, 0))

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), s.account(1)))

	s.noBrief(contractDay, 1)
	s.Empty(s.sheet(contractDay, 1).Progress)
	s.True(s.sheet(contractDay, 1).Paid.Empty())
	s.Zero(s.paidInAll(1))
	s.Equal(NothingDone(), s.done(1))
	_, err := s.store.Brief(s.T().Context(), contractDay, s.account(2))
	s.Require().NoError(err, "another account keeps its day")
	s.NotEmpty(s.sheet(contractDay, 2).Progress)
}
