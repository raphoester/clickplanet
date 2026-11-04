//go:build testing

package stats

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var contractAt = time.Date(2026, 9, 17, 23, 30, 0, 123_456_000, time.UTC)

func (s *StoreContractSuite) TestAnUnknownAccountHasNoStats() {
	_, err := s.store.Stats(s.T().Context(), accounts.AccountID{15: 1})
	s.Require().ErrorIs(err, ErrNoStats)
}

func (s *StoreContractSuite) TestAForgottenAccountLosesItsStatsAndTheOthersKeepTheirs() {
	s.recordTake(1, 0, contractAt)
	s.recordTake(2, 1, contractAt)

	s.Require().NoError(s.store.Forget(s.T().Context(), accounts.AccountID{15: 1}))

	_, err := s.store.Stats(s.T().Context(), accounts.AccountID{15: 1})
	s.Require().ErrorIs(err, ErrNoStats)
	s.Equal(uint64(1), s.stats(2).TilesTaken())
}

func (s *StoreContractSuite) TestForgettingAnUnknownAccountIsNotAnError() {
	s.Require().NoError(s.store.Forget(s.T().Context(), accounts.AccountID{15: 9}))
}

func (s *StoreContractSuite) recordTake(account byte, position uint64, at time.Time) {
	s.Require().NoError(s.store.Record(s.T().Context(), accounts.AccountID{15: account}, TakeAt(position, 0, at)))
}

func (s *StoreContractSuite) record(account byte, change func(Stats) Stats) {
	s.Require().NoError(s.store.Record(s.T().Context(), accounts.AccountID{15: account}, change))
}

func (s *StoreContractSuite) stats(account byte) Stats {
	stats, err := s.store.Stats(s.T().Context(), accounts.AccountID{15: account})
	s.Require().NoError(err)
	return stats
}

func (s *StoreContractSuite) TestEachTakeIsCountedByTheDomainsRule() {
	s.recordTake(1, 0, contractAt)
	s.recordTake(1, 1, contractAt.Add(time.Minute))
	s.recordTake(1, 2, contractAt.Add(time.Hour))

	want := NewStats(accounts.AccountID{15: 1}).WithTake(0, 0, contractAt).WithTake(1, 0, contractAt.Add(time.Minute)).
		WithTake(2, 0, contractAt.Add(time.Hour))
	s.Equal(want, s.stats(1))
	s.Equal(uint32(2), s.stats(1).Streak().Days(), "the third take is past UTC midnight")
}

func (s *StoreContractSuite) TestTakesOfOneAccountDoNotCountOnAnother() {
	s.recordTake(1, 0, contractAt)
	s.recordTake(2, 1, contractAt)
	s.recordTake(2, 2, contractAt)

	s.Equal(uint64(1), s.stats(1).TilesTaken())
	s.Equal(uint64(2), s.stats(2).TilesTaken())
}

func (s *StoreContractSuite) TestEachMessageIsCountedAndStartsNoStreak() {
	s.record(1, MessageAt(contractAt))
	s.record(1, MessageAt(contractAt.Add(time.Second)))

	s.Equal(NewStats(accounts.AccountID{15: 1}).WithMessage(contractAt).WithMessage(contractAt.Add(time.Second)), s.stats(1))
}

func (s *StoreContractSuite) TestMessagesAndTakesAddUpOnOneAccount() {
	s.record(1, MessageAt(contractAt))
	s.recordTake(1, 0, contractAt)
	s.record(1, MessageAt(contractAt.Add(time.Second)))
	s.record(2, MessageAt(contractAt))

	s.Equal(NewStats(accounts.AccountID{15: 1}).WithMessage(contractAt).WithTake(0, 0, contractAt).
		WithMessage(contractAt.Add(time.Second)), s.stats(1))
	s.Equal(uint64(1), s.stats(2).MessagesSent())
}

func (s *StoreContractSuite) TestEveryCounterReadsBack() {
	for _, change := range []func(Stats) Stats{
		BombAt(3), BonusAt(contractAt), RightAnswerAt(contractAt), BombAt(8), FortifyAt(5, 137), MissionsPaid(4),
		MissionsPaid(2),
	} {
		s.record(1, change)
	}

	s.Equal(StatsSnapshot{
		Account: accounts.AccountID{15: 1}, BombsDropped: 2, TilesFortified: 137, BonusesClaimed: 1, RightAnswers: 1,
		MissionsDone: 4,
		Counted:      Counted{BombsBelow: 9, FortifiesBelow: 6, BonusesUntil: contractAt, AnswersUntil: contractAt},
	}, s.stats(1).TakeSnapshot())
}

func (s *StoreContractSuite) TestAFactToldAgainAfterItWasKeptIsNotCountedAgain() {
	for range 2 {
		for _, change := range []func(Stats) Stats{
			TakeAt(4, 0, contractAt), MessageAt(contractAt), BombAt(5), FortifyAt(6, 60), BonusAt(contractAt),
			RightAnswerAt(contractAt), MissionsPaid(3),
		} {
			s.record(1, change)
		}
	}

	s.Equal(StatsSnapshot{
		Account: accounts.AccountID{15: 1}, TilesTaken: 1, Streak: StreakOf(1, DayOf(contractAt)), StreakBest: 1, MessagesSent: 1,
		BombsDropped: 1, TilesFortified: 60, BonusesClaimed: 1, RightAnswers: 1, MissionsDone: 3,
		Counted: Counted{
			TakesBelow: TakeMark{Position: 4, Index: 1}, BombsBelow: 6, FortifiesBelow: 7,
			MessagesUntil: contractAt, BonusesUntil: contractAt, AnswersUntil: contractAt,
		},
	}, s.stats(1).TakeSnapshot())
}

func (s *StoreContractSuite) TestEachTakeOfASpreadIsCountedOnceThoughTheyShareAPosition() {
	for range 2 {
		for index := range uint32(3) {
			s.record(1, TakeAt(4, index, contractAt))
		}
	}

	s.Equal(uint64(3), s.stats(1).TilesTaken())
	s.Equal(TakeMark{Position: 4, Index: 3}, s.stats(1).TakeSnapshot().Counted.TakesBelow)
}

func (s *StoreContractSuite) TestStatsArePagedInAccountOrderAfterTheCursor() {
	for _, account := range []byte{3, 1, 4, 2} {
		s.recordTake(account, 0, contractAt)
	}

	first, err := s.store.StatsAfter(s.T().Context(), accounts.AccountID{}, 3)
	s.Require().NoError(err)
	rest, err := s.store.StatsAfter(s.T().Context(), first[len(first)-1].Account(), 3)
	s.Require().NoError(err)

	accountsOf := func(page []Stats) []accounts.AccountID {
		accounts := make([]accounts.AccountID, 0, len(page))
		for _, stats := range page {
			accounts = append(accounts, stats.Account())
		}
		return accounts
	}
	s.Equal([]accounts.AccountID{{15: 1}, {15: 2}, {15: 3}}, accountsOf(first))
	s.Equal([]accounts.AccountID{{15: 4}}, accountsOf(rest))
	s.Equal(s.stats(4), rest[0])
}
