package unmute_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation/inmemory_moderation_storage"
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
	operator = moderation.OperatorOf(messages.AccountID{15: 9}, "")
)

type testSuite struct {
	suite.Suite

	mutes   *inmemory_mute_storage.Storage
	history *inmemory_moderation_storage.Storage
}

func (s *testSuite) SetupTest() {
	s.mutes = inmemory_mute_storage.New()
	s.history = inmemory_moderation_storage.New()
}

func (s *testSuite) kept(id byte, account messages.AccountID, scope mutes.Scope, at time.Time, duration time.Duration) mutes.Mute {
	mute := mutes.NewMute(mutes.MuteID{15: id}, mutes.CallerOf(account, scope), mutes.Chat, at, duration)
	s.Require().NoError(s.mutes.Save(context.Background(), mute))
	return mute
}

func (s *testSuite) unmute(store unmute_usecase.Store, account messages.AccountID) (time.Time, error) {
	return s.unmuteOf(store, account, mutes.Chat)
}

func (s *testSuite) unmuteOf(store unmute_usecase.Store, account messages.AccountID, kind mutes.Kind) (time.Time, error) {
	return unmute_usecase.New(cpatomic.FakeTransactioner{}, store, s.history, cptime.NewFixedClock(now)).
		Execute(context.Background(), unmute_usecase.In{Account: account, Kind: kind, Reason: "served", By: operator})
}

func (s *testSuite) TestAnUnmuteIsRecordedWithItsReasonAndWhoGaveIt() {
	s.kept(1, bully, home, now.Add(-time.Minute), time.Hour)

	_, err := s.unmuteOf(s.mutes, bully, mutes.Chat)
	s.Require().NoError(err)

	s.Equal([]moderation.Snapshot{{
		Account: bully, At: now, Kind: moderation.Unmuted, MuteKind: mutes.Chat, Reason: "served", By: operator,
	}}, s.history.Kept(), "one action, however many mutes it lifts")
}

func (s *testSuite) TestAnInvalidReasonIsRefusedAndNothingIsLifted() {
	running := s.kept(1, bully, home, now.Add(-time.Minute), time.Hour)

	_, err := unmute_usecase.New(cpatomic.FakeTransactioner{}, s.mutes, s.history, cptime.NewFixedClock(now)).
		Execute(context.Background(), unmute_usecase.In{Account: bully, Reason: string([]byte{0xff})})

	s.Require().ErrorIs(err, moderation.ErrInvalidReason)
	s.Equal([]mutes.Mute{running}, s.mutes.Kept())
	s.Empty(s.history.Kept())
}

func (s *testSuite) TestAnUnmuteTheHistoryCannotRecordFails() {
	s.kept(1, bully, home, now.Add(-time.Minute), time.Hour)
	s.history.FailWith(errors.New("postgres is down"))

	_, err := s.unmute(s.mutes, bully)

	s.Require().Error(err, "the transaction keeps neither")
	s.NotErrorIs(err, mutes.ErrNotMuted)
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
	s.NoError(book.MuteError(context.Background(), mutes.CallerOf(bully, home), mutes.Chat))
	s.NoError(book.MuteError(context.Background(), mutes.CallerOf(newGuest, home), mutes.Chat))
}

func (s *testSuite) TestAMuteOfAnotherKindStays() {
	gifs := mutes.NewMute(mutes.MuteID{15: 1}, mutes.CallerOf(bully, home), mutes.Gifs, now.Add(-time.Minute), time.Hour)
	s.Require().NoError(s.mutes.Save(context.Background(), gifs))
	chat := s.kept(2, bully, home, now.Add(-time.Minute), 24*time.Hour)

	until, err := s.unmuteOf(s.mutes, bully, mutes.Gifs)
	s.Require().NoError(err)

	s.Equal(gifs.Until(), until)
	s.ElementsMatch([]mutes.Mute{gifs.Lifted(now), chat}, s.mutes.Kept())
}

func (s *testSuite) TestAMuteGivenToAnotherAccountOnItsNetworkStays() {
	household := s.kept(1, newGuest, home, now.Add(-time.Minute), time.Hour)
	s.kept(2, bully, home, now.Add(-time.Minute), time.Hour)

	_, err := s.unmute(s.mutes, bully)
	s.Require().NoError(err)

	s.Contains(s.mutes.Kept(), household)
	s.Error(mutes.NewBook(s.mutes, cptime.NewFixedClock(now)).MuteError(context.Background(), mutes.CallerOf(bully, home), mutes.Chat),
		"the household's mute still holds it")
}

func (s *testSuite) TestAnAccountWithNoRunningMuteIsNotMuted() {
	ended := s.kept(1, bully, home, now.Add(-2*time.Hour), time.Hour)
	s.kept(2, newGuest, home, now.Add(-time.Minute), time.Hour)

	_, err := s.unmute(s.mutes, bully)

	s.Require().ErrorIs(err, mutes.ErrNotMuted)
	s.Contains(s.mutes.Kept(), ended)
	s.Empty(s.history.Kept(), "an unmute that lifts nothing is no action")
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
