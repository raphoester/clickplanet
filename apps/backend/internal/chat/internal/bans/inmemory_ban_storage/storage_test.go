package inmemory_ban_storage_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/inmemory_ban_storage"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	persistence *inmemory_ban_storage.MemoryPersistence
	storage     *inmemory_ban_storage.Storage
}

var noon = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)

func (s *testSuite) SetupTest() {
	s.persistence = inmemory_ban_storage.NewMemoryPersistence()
	s.storage = inmemory_ban_storage.New(s.persistence, slog.New(slog.DiscardHandler))
}

func ban(tag string, at time.Time) bans.Ban {
	return bans.Ban{AuthorTag: tag, BannedAt: at, Reason: "spam"}
}

func (s *testSuite) TestAnUnknownTagIsNotBanned() {
	s.False(s.storage.Banned("a1b2c3"))
}

func (s *testSuite) TestABannedTagIsBannedAndRecorded() {
	stored, err := s.storage.Ban(context.Background(), ban("a1b2c3", noon))
	s.Require().NoError(err)

	s.Equal(ban("a1b2c3", noon), stored)
	s.True(s.storage.Banned("a1b2c3"))
	s.Equal([]bans.Ban{ban("a1b2c3", noon)}, s.persistence.Stored())
}

func (s *testSuite) TestABanThatCannotBeRecordedIsNotEnforced() {
	s.persistence.FailWith(assert.AnError)

	_, err := s.storage.Ban(context.Background(), ban("a1b2c3", noon))

	s.Require().ErrorIs(err, assert.AnError)
	s.False(s.storage.Banned("a1b2c3"), "a ban no table remembers would not survive a restart")
}

func (s *testSuite) TestBanningTwiceKeepsWhenTheMemberWasFirstSilenced() {
	_, err := s.storage.Ban(context.Background(), ban("a1b2c3", noon))
	s.Require().NoError(err)

	again, err := s.storage.Ban(context.Background(),
		bans.Ban{AuthorTag: "a1b2c3", BannedAt: noon.Add(time.Hour), Reason: "still spam"})
	s.Require().NoError(err)

	s.Equal(noon, again.BannedAt)
	s.Equal("still spam", again.Reason)
	s.Require().Len(s.persistence.Stored(), 1)
	s.Equal(noon, s.persistence.Stored()[0].BannedAt)
}

func (s *testSuite) TestUnbanningLiftsItEverywhere() {
	_, err := s.storage.Ban(context.Background(), ban("a1b2c3", noon))
	s.Require().NoError(err)

	s.Require().NoError(s.storage.Unban(context.Background(), "a1b2c3"))

	s.False(s.storage.Banned("a1b2c3"))
	s.Empty(s.persistence.Stored())
}

func (s *testSuite) TestUnbanningSomebodyNobodyBanned() {
	err := s.storage.Unban(context.Background(), "a1b2c3")

	s.Require().ErrorIs(err, bans.ErrNotBanned)
}

func (s *testSuite) TestAnUnbanThatCannotBeRecordedStaysBanned() {
	_, err := s.storage.Ban(context.Background(), ban("a1b2c3", noon))
	s.Require().NoError(err)
	s.persistence.FailWith(assert.AnError)

	s.Require().ErrorIs(s.storage.Unban(context.Background(), "a1b2c3"), assert.AnError)
	s.True(s.storage.Banned("a1b2c3"))
}

func (s *testSuite) TestTheBansAreListedNewestFirst() {
	for _, each := range []bans.Ban{ban("aaaaaa", noon), ban("cccccc", noon.Add(time.Hour)), ban("bbbbbb", noon)} {
		_, err := s.storage.Ban(context.Background(), each)
		s.Require().NoError(err)
	}

	s.Equal([]string{"cccccc", "aaaaaa", "bbbbbb"}, tags(s.storage.All()))
}

func (s *testSuite) TestTheSetIsFilledFromWhatWasKept() {
	persistence := inmemory_ban_storage.NewMemoryPersistence(ban("a1b2c3", noon))
	storage := inmemory_ban_storage.New(persistence, slog.New(slog.DiscardHandler))

	s.Require().NoError(storage.Load(context.Background()))

	s.True(storage.Banned("a1b2c3"))
}

func (s *testSuite) TestAFailedLoadIsReported() {
	s.persistence.FailWith(assert.AnError)

	s.Require().ErrorIs(s.storage.Load(context.Background()), assert.AnError)
}

func tags(all []bans.Ban) []string {
	names := make([]string, 0, len(all))
	for _, each := range all {
		names = append(names, each.AuthorTag)
	}
	return names
}

func TestTheSetIsSafeToReadWhileItIsWritten(t *testing.T) {
	storage := inmemory_ban_storage.New(inmemory_ban_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))

	failed := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 100 {
			if _, err := storage.Ban(t.Context(), ban("a1b2c3", noon)); err != nil {
				failed <- err
				return
			}
		}
	}()

	for range 100 {
		storage.Banned("a1b2c3")
		storage.All()
	}

	<-done
	close(failed)
	require.NoError(t, <-failed)
}
