package unmute_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/inmemory_mute_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/unmute_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	now      = time.Date(2026, 10, 30, 20, 0, 0, 0, time.UTC)
	bully    = messages.AccountID{15: 1}
	newGuest = messages.AccountID{15: 2}
	home     = mutes.Scope("2a01:e0a:1:2::/64")
)

type testSuite struct {
	suite.Suite

	mutes *inmemory_mute_storage.Storage
}

func (s *testSuite) SetupTest() {
	s.mutes = inmemory_mute_storage.New()
}

func (s *testSuite) kept(id byte, account messages.AccountID, scope mutes.Scope, at time.Time, duration time.Duration) mutes.Mute {
	mute := mutes.NewMute(mutes.MuteID{15: id}, mutes.CallerOf(account, scope), at, duration)
	s.Require().NoError(s.mutes.Save(context.Background(), mute))
	return mute
}

func (s *testSuite) unmute(store unmute_usecase.Store, account messages.AccountID) (time.Time, error) {
	return unmute_usecase.New(cpatomic.FakeTransactioner{}, store, cptime.NewFixedClock(now)).
		Execute(context.Background(), account)
}

func (s *testSuite) TestEveryRunningMuteOfTheAccountEndsNowWithItsNetwork() {
	hour := s.kept(1, bully, home, now.Add(-time.Minute), time.Hour)
	day := s.kept(2, bully, mutes.NoScope, now.Add(-time.Minute), 24*time.Hour)
	ended := s.kept(3, bully, home, now.Add(-2*time.Hour), time.Hour)

	until, err := s.unmute(s.mutes, bully)
	s.Require().NoError(err)

	s.Equal(day.Until(), until, "the end of the mute that would have ended last")
	s.ElementsMatch([]mutes.Mute{hour.Lifted(now), day.Lifted(now), ended}, s.mutes.Kept(),
		"a mute that had ended is left as it was")
	book := mutes.NewBook(s.mutes, cptime.NewFixedClock(now))
	s.NoError(book.MuteError(context.Background(), mutes.CallerOf(bully, home)))
	s.NoError(book.MuteError(context.Background(), mutes.CallerOf(newGuest, home)))
}

func (s *testSuite) TestAMuteGivenToAnotherAccountOnItsNetworkStays() {
	household := s.kept(1, newGuest, home, now.Add(-time.Minute), time.Hour)
	s.kept(2, bully, home, now.Add(-time.Minute), time.Hour)

	_, err := s.unmute(s.mutes, bully)
	s.Require().NoError(err)

	s.Contains(s.mutes.Kept(), household)
	s.Error(mutes.NewBook(s.mutes, cptime.NewFixedClock(now)).MuteError(context.Background(), mutes.CallerOf(bully, home)),
		"the household's mute still holds it")
}

func (s *testSuite) TestAnAccountWithNoRunningMuteIsNotMuted() {
	ended := s.kept(1, bully, home, now.Add(-2*time.Hour), time.Hour)
	s.kept(2, newGuest, home, now.Add(-time.Minute), time.Hour)

	_, err := s.unmute(s.mutes, bully)

	s.Require().ErrorIs(err, mutes.ErrNotMuted)
	s.Contains(s.mutes.Kept(), ended)
}

func (s *testSuite) TestNoAccountIsRefused() {
	_, err := s.unmute(s.mutes, messages.NoAccount)

	s.Require().ErrorIs(err, mutes.ErrNoAccount)
}

func (s *testSuite) TestAMuteThatCannotBeLiftedFails() {
	s.kept(1, bully, home, now.Add(-time.Minute), time.Hour)

	_, err := s.unmute(failingSaver{s.mutes}, bully)

	s.Require().Error(err)
	s.NotErrorIs(err, mutes.ErrNotMuted)
}

type failingSaver struct {
	*inmemory_mute_storage.Storage
}

func (failingSaver) Save(context.Context, mutes.Mute) error { return errors.New("postgres is down") }
