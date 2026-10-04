//go:build testing

package takes

import (
	"context"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Players interface {
	Stats(ctx context.Context, account players.AccountID) (players.Stats, error)
	RecordTake(ctx context.Context, account players.AccountID, at time.Time) error
	RecordMessage(ctx context.Context, account players.AccountID) error
	DeleteAccount(ctx context.Context, account players.AccountID) error
}

type StoreContractSuite struct {
	suite.Suite

	NewStore func() (Store, Players)

	store   Store
	players Players
}

func (s *StoreContractSuite) SetupTest() {
	s.store, s.players = s.NewStore()
}

var (
	contractAda = players.AccountID{15: 1}
	contractBob = players.AccountID{15: 2}
	contractAt  = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
)

func (s *StoreContractSuite) batch(from Position, takes ...Take) Batch {
	next := from
	if len(takes) > 0 {
		next = takes[len(takes)-1].position + 1
	}
	batch, err := BatchOf(from, next, takes)
	s.Require().NoError(err)
	return batch
}

func (s *StoreContractSuite) stats(account players.AccountID) players.Stats {
	stats, err := s.players.Stats(s.T().Context(), account)
	s.Require().NoError(err)
	return stats
}

func (s *StoreContractSuite) position() Position {
	position, err := s.store.Position(s.T().Context())
	s.Require().NoError(err)
	return position
}

func (s *StoreContractSuite) busCounted(account players.AccountID, at time.Time) {
	s.Require().NoError(s.players.RecordTake(s.T().Context(), account, at))
}

func (s *StoreContractSuite) TestNothingIsCountedBeforeTheStart() {
	_, err := s.store.Position(s.T().Context())
	s.Require().ErrorIs(err, ErrNotStarted)

	s.Require().ErrorIs(s.store.Count(s.T().Context(), s.batch(0, TakeOf(0, contractAda, "fr", contractAt, false))), ErrNotStarted)
	_, err = s.players.Stats(s.T().Context(), contractAda)
	s.Require().ErrorIs(err, players.ErrNoStats)

	_, err = s.store.Rewind(s.T().Context())
	s.Require().ErrorIs(err, ErrNotStarted)
}

func (s *StoreContractSuite) TestTheCountBeginsAtItsStartOnce() {
	s.Require().NoError(s.store.Begin(s.T().Context(), 5))
	s.Equal(Position(5), s.position())

	s.Require().ErrorIs(s.store.Begin(s.T().Context(), 9), ErrStarted)
	s.Equal(Position(5), s.position())
}

func (s *StoreContractSuite) TestABatchIsCountedByTheDomainsRuleAndMovesThePosition() {
	s.Require().NoError(s.store.Begin(s.T().Context(), 0))

	s.Require().NoError(s.store.Count(s.T().Context(), s.batch(0,
		TakeOf(0, contractAda, "fr", contractAt, false),
		TakeOf(1, contractBob, "de", contractAt, false),
		TakeOf(3, contractAda, "fr", contractAt.Add(24*time.Hour), false),
	)))

	s.Equal(players.NewStats(contractAda).WithTake(contractAt).WithTake(contractAt.Add(24*time.Hour)), s.stats(contractAda))
	s.Equal(players.NewStats(contractBob).WithTake(contractAt), s.stats(contractBob))
	s.Equal(Position(4), s.position())
}

func (s *StoreContractSuite) TestATakeThatDoesNotCountIsPassedWithoutCounting() {
	s.Require().NoError(s.store.Begin(s.T().Context(), 0))

	s.Require().NoError(s.store.Count(s.T().Context(), s.batch(0,
		TakeOf(0, contractAda, "", contractAt, false),
		TakeOf(1, contractAda, "fr", contractAt, true),
	)))

	_, err := s.players.Stats(s.T().Context(), contractAda)
	s.Require().ErrorIs(err, players.ErrNoStats)
	s.Equal(Position(2), s.position())
}

func (s *StoreContractSuite) TestEntriesThatAreNotTakesMoveThePositionPastThem() {
	s.Require().NoError(s.store.Begin(s.T().Context(), 0))
	batch, err := BatchOf(0, 5, []Take{TakeOf(2, contractAda, "fr", contractAt, false)})
	s.Require().NoError(err)

	s.Require().NoError(s.store.Count(s.T().Context(), batch))

	s.Equal(uint64(1), s.stats(contractAda).TilesTaken())
	s.Equal(Position(5), s.position())
}

func (s *StoreContractSuite) TestABatchCountedTwiceCountsOnce() {
	s.Require().NoError(s.store.Begin(s.T().Context(), 0))
	batch := s.batch(0, TakeOf(0, contractAda, "fr", contractAt, false))
	s.Require().NoError(s.store.Count(s.T().Context(), batch))

	s.Require().ErrorIs(s.store.Count(s.T().Context(), batch), ErrMoved)

	s.Equal(uint64(1), s.stats(contractAda).TilesTaken())
	s.Equal(Position(1), s.position())
}

func (s *StoreContractSuite) TestCountingKeepsTheMessagesSent() {
	s.Require().NoError(s.players.RecordMessage(s.T().Context(), contractAda))
	s.Require().NoError(s.store.Begin(s.T().Context(), 0))

	s.Require().NoError(s.store.Count(s.T().Context(), s.batch(0, TakeOf(0, contractAda, "fr", contractAt, false))))
	s.Require().NoError(s.players.RecordMessage(s.T().Context(), contractAda))

	s.Equal(players.NewStats(contractAda).WithMessage().WithTake(contractAt).WithMessage(), s.stats(contractAda))
}

func (s *StoreContractSuite) TestARewindPutsBackTheStatsAsTheyWereAtTheStart() {
	s.busCounted(contractAda, contractAt.Add(-24*time.Hour))
	s.busCounted(contractAda, contractAt.Add(-time.Hour))
	s.Require().NoError(s.players.RecordMessage(s.T().Context(), contractAda))
	baseline := s.stats(contractAda)
	s.Require().NoError(s.store.Begin(s.T().Context(), 10))
	s.Require().NoError(s.store.Count(s.T().Context(), s.batch(10,
		TakeOf(10, contractAda, "fr", contractAt.Add(24*time.Hour), false),
		TakeOf(11, contractBob, "de", contractAt, false),
	)))
	s.Require().NoError(s.players.RecordMessage(s.T().Context(), contractAda))

	start, err := s.store.Rewind(s.T().Context())

	s.Require().NoError(err)
	s.Equal(Position(10), start)
	s.Equal(Position(10), s.position())
	s.Equal(baseline.WithMessage(), s.stats(contractAda), "the takes as they were, the messages as they are")
	s.Equal(players.NewStats(contractBob), s.stats(contractBob), "an account with no take before the start has none")
}

func (s *StoreContractSuite) TestCountingTheSameTakesAgainAfterARewindGivesTheSameStats() {
	s.busCounted(contractAda, contractAt.Add(-24*time.Hour))
	s.Require().NoError(s.store.Begin(s.T().Context(), 0))
	batch := s.batch(0,
		TakeOf(0, contractAda, "fr", contractAt, false),
		TakeOf(1, contractBob, "de", contractAt, false),
		TakeOf(2, contractAda, "fr", contractAt.Add(24*time.Hour), false),
	)
	s.Require().NoError(s.store.Count(s.T().Context(), batch))
	live := []players.Stats{s.stats(contractAda), s.stats(contractBob)}

	_, err := s.store.Rewind(s.T().Context())
	s.Require().NoError(err)
	s.Require().NoError(s.store.Count(s.T().Context(), batch))

	rebuilt := []players.Stats{s.stats(contractAda), s.stats(contractBob)}
	s.Equal(live, rebuilt)
}

func (s *StoreContractSuite) TestARebuildPastATakeRevertedSinceCountsOneTileLess() {
	s.Require().NoError(s.store.Begin(s.T().Context(), 0))
	s.Require().NoError(s.store.Count(s.T().Context(), s.batch(0,
		TakeOf(0, contractAda, "fr", contractAt, false),
		TakeOf(1, contractAda, "fr", contractAt, false),
	)))
	s.Equal(uint64(2), s.stats(contractAda).TilesTaken(), "the live count counts what it read")

	_, err := s.store.Rewind(s.T().Context())
	s.Require().NoError(err)
	s.Require().NoError(s.store.Count(s.T().Context(), s.batch(0,
		TakeOf(0, contractAda, "fr", contractAt, false),
		TakeOf(1, contractAda, "fr", contractAt, true),
	)))

	s.Equal(uint64(1), s.stats(contractAda).TilesTaken())
}

func (s *StoreContractSuite) TestARewindNeverBringsBackADeletedAccount() {
	s.busCounted(contractAda, contractAt)
	s.Require().NoError(s.store.Begin(s.T().Context(), 0))
	s.Require().NoError(s.players.DeleteAccount(s.T().Context(), contractAda))

	_, err := s.store.Rewind(s.T().Context())
	s.Require().NoError(err)

	_, err = s.players.Stats(s.T().Context(), contractAda)
	s.Require().ErrorIs(err, players.ErrNoStats)
}

func (s *StoreContractSuite) TestADeletedAccountLeavesNoBaseline() {
	s.busCounted(contractAda, contractAt)
	s.busCounted(contractBob, contractAt)
	s.Require().NoError(s.store.Begin(s.T().Context(), 0))

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), contractAda))
	_, err := s.store.Rewind(s.T().Context())
	s.Require().NoError(err)

	s.Zero(s.stats(contractAda).TilesTaken())
	s.Equal(uint64(1), s.stats(contractBob).TilesTaken())
}

func (s *StoreContractSuite) TestDeletingAnAccountWithNoBaselineIsNotAnError() {
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), contractAda))
}
