//go:build testing

package raids

import (
	"context"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
)

type StorageContractSuite struct {
	suite.Suite

	NewStorage func() Storage

	storage Storage
}

var contractStart = time.Date(2026, 11, 7, 12, 0, 0, 123_456_000, time.UTC)

func (s *StorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

func (s *StorageContractSuite) started(at time.Time, duration time.Duration, reason string) RaidMode {
	mode, err := Start(at, duration, "")
	s.Require().NoError(err)
	mode.reason = moderation.Reason(reason)
	s.Require().NoError(s.storage.Save(context.Background(), mode))
	return mode
}

func (s *StorageContractSuite) TestNoneSavedIsNoRaidMode() {
	_, err := s.storage.Current(context.Background())

	s.Require().ErrorIs(err, ErrNoRaidMode)
}

func (s *StorageContractSuite) TestARaidModeSavedIsReadBackToTheMicrosecond() {
	mode := s.started(contractStart, time.Hour, "a raid of guests")

	read, err := s.storage.Current(context.Background())

	s.Require().NoError(err)
	s.Equal(mode.TakeSnapshot(), read.TakeSnapshot())
}

func (s *StorageContractSuite) TestARaidModeSavedReplacesTheOneBefore() {
	s.started(contractStart, time.Hour, "first")
	second := s.started(contractStart.Add(10*time.Minute), 24*time.Hour, "second")

	read, err := s.storage.Current(context.Background())

	s.Require().NoError(err)
	s.Equal(second.TakeSnapshot(), read.TakeSnapshot(), "one at a time")
}

func (s *StorageContractSuite) TestAnEndedRaidModeIsKeptEnded() {
	mode := s.started(contractStart, time.Hour, "")
	s.Require().NoError(s.storage.Save(context.Background(), mode.Ended(contractStart.Add(time.Minute))))

	read, err := s.storage.Current(context.Background())

	s.Require().NoError(err)
	s.False(read.RunsAt(contractStart.Add(time.Minute)))
	s.Equal(contractStart.Add(time.Minute), read.TakeSnapshot().EndsAt)
}
