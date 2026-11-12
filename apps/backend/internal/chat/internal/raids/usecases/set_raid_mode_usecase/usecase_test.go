package set_raid_mode_usecase_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids/inmemory_raid_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids/usecases/set_raid_mode_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var now = time.Date(2026, 11, 7, 20, 0, 0, 0, time.UTC)

type testSuite struct {
	suite.Suite

	store   *inmemory_raid_storage.Storage
	clock   *cptime.FixedClock
	useCase *set_raid_mode_usecase.UseCase
}

func (s *testSuite) SetupTest() {
	s.store = inmemory_raid_storage.New()
	s.clock = cptime.NewFixedClock(now)
	s.useCase = set_raid_mode_usecase.New(s.store, s.clock)
}

func (s *testSuite) set(duration time.Duration, reason string) (raids.RaidMode, bool, error) {
	return s.useCase.Execute(s.T().Context(), set_raid_mode_usecase.In{Duration: duration, Reason: reason})
}

func (s *testSuite) kept() raids.Snapshot {
	mode, err := s.store.Current(s.T().Context())
	s.Require().NoError(err)
	return mode.TakeSnapshot()
}

func (s *testSuite) TestARaidModeStartsNowForItsDurationWithItsReason() {
	mode, running, err := s.set(time.Hour, "  a raid of guests  ")

	s.Require().NoError(err)
	s.True(running)
	want := raids.Snapshot{StartedAt: now, EndsAt: now.Add(time.Hour), Reason: "a raid of guests"}
	s.Equal(want, mode.TakeSnapshot())
	s.Equal(want, s.kept())
}

func (s *testSuite) TestSetAgainWhileItRunsItStartsAgainFromNow() {
	_, _, err := s.set(time.Hour, "first")
	s.Require().NoError(err)
	s.clock.Advance(30 * time.Minute)

	_, _, err = s.set(10*time.Minute, "shorter")

	s.Require().NoError(err)
	s.Equal(raids.Snapshot{
		StartedAt: now.Add(30 * time.Minute), EndsAt: now.Add(40 * time.Minute), Reason: "shorter",
	}, s.kept())
}

func (s *testSuite) TestNoDurationEndsTheRaidModeNow() {
	_, _, err := s.set(time.Hour, "a raid")
	s.Require().NoError(err)
	s.clock.Advance(time.Minute)

	_, running, err := s.set(0, "it is over")

	s.Require().NoError(err)
	s.False(running)
	s.Equal(raids.Snapshot{StartedAt: now, EndsAt: now.Add(time.Minute), Reason: "a raid"}, s.kept(),
		"it ends when it was ended, and keeps why it started")
}

func (s *testSuite) TestEndingWhenNoneRunsChangesNothing() {
	_, running, err := s.set(0, "")
	s.Require().NoError(err)
	s.False(running)
	_, err = s.store.Current(s.T().Context())
	s.Require().ErrorIs(err, raids.ErrNoRaidMode, "nothing kept")

	_, _, err = s.set(time.Hour, "")
	s.Require().NoError(err)
	s.clock.Advance(2 * time.Hour)
	_, _, err = s.set(0, "")
	s.Require().NoError(err)
	s.Equal(now.Add(time.Hour), s.kept().EndsAt, "one that ended by itself keeps its end")
}

func (s *testSuite) TestADurationPastADayIsRefusedAndNothingIsKept() {
	_, _, err := s.set(25*time.Hour, "")

	s.Require().ErrorIs(err, raids.ErrInvalidDuration)
	_, err = s.store.Current(s.T().Context())
	s.Require().ErrorIs(err, raids.ErrNoRaidMode)
}

func (s *testSuite) TestAReasonTooLongIsRefused() {
	_, _, err := s.set(time.Hour, strings.Repeat("a", moderation.MaxReasonLength+1))

	s.Require().ErrorIs(err, moderation.ErrInvalidReason)
}
