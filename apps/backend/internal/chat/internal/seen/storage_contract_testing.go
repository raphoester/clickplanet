//go:build testing

package seen

import (
	"context"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type StorageContractSuite struct {
	suite.Suite

	NewStorage func() Storage

	storage Storage
}

var (
	contractStart = time.Date(2026, 10, 4, 12, 0, 0, 123_000_000, time.UTC)
	contractAda   = messages.AccountID{15: 1}
	contractBob   = messages.AccountID{15: 2}
)

func (s *StorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

func (s *StorageContractSuite) seenUntil(account messages.AccountID) time.Time {
	until, err := s.storage.SeenUntil(context.Background(), account)
	s.Require().NoError(err)
	return until
}

func (s *StorageContractSuite) save(account messages.AccountID, until time.Time) {
	s.Require().NoError(s.storage.SaveSeen(context.Background(), account, until))
}

func (s *StorageContractSuite) TestAnAccountWithNoMarkHasSeenNothing() {
	s.True(s.seenUntil(contractAda).IsZero())
}

func (s *StorageContractSuite) TestAMarkReadsBackToTheMillisecond() {
	s.save(contractAda, contractStart)

	until := s.seenUntil(contractAda)
	s.True(contractStart.Equal(until), "until %s, want %s", until, contractStart)
}

func (s *StorageContractSuite) TestAMarkOnlyMovesForward() {
	s.save(contractAda, contractStart.Add(time.Hour))
	s.save(contractAda, contractStart)

	s.True(contractStart.Add(time.Hour).Equal(s.seenUntil(contractAda)))

	s.save(contractAda, contractStart.Add(2*time.Hour))

	s.True(contractStart.Add(2 * time.Hour).Equal(s.seenUntil(contractAda)))
}

func (s *StorageContractSuite) TestEachAccountHasItsOwnMark() {
	s.save(contractAda, contractStart)

	s.True(s.seenUntil(contractBob).IsZero())
}

func (s *StorageContractSuite) TestADeletedMarkIsGoneAndOnlyThatOne() {
	s.save(contractAda, contractStart)
	s.save(contractBob, contractStart)

	s.Require().NoError(s.storage.DeleteSeen(context.Background(), contractAda))

	s.True(s.seenUntil(contractAda).IsZero())
	s.True(contractStart.Equal(s.seenUntil(contractBob)))
}

func (s *StorageContractSuite) TestDeletingNoMarkIsNotAnError() {
	s.NoError(s.storage.DeleteSeen(context.Background(), contractAda))
}
