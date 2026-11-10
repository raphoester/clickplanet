package mute_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/inmemory_message_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation/inmemory_moderation_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/inmemory_mute_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/mute_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	now    = time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC)
	bully  = messages.AccountID{15: 1}
	quiet  = messages.AccountID{15: 2}
	claude = moderation.OperatorOf(messages.AccountID{15: 9}, "Claude")
)

type testSuite struct {
	suite.Suite

	mutes    *inmemory_mute_storage.Storage
	messages *inmemory_message_storage.Storage
	history  *inmemory_moderation_storage.Storage
	useCase  *mute_usecase.UseCase
}

func (s *testSuite) SetupTest() {
	s.mutes = inmemory_mute_storage.New()
	s.messages = inmemory_message_storage.New()
	s.history = inmemory_moderation_storage.New()
	s.useCase = s.build(s.mutes, &mutes.SequentialIDs{})
}

func (s *testSuite) build(saver mute_usecase.Saver, ids mutes.IDProvider) *mute_usecase.UseCase {
	return mute_usecase.New(cpatomic.FakeTransactioner{}, saver, s.history, s.messages, ids, cptime.NewFixedClock(now))
}

func (s *testSuite) posted(account messages.AccountID, ip string, id string) {
	message := messages.Restored(s.T(), messages.Snapshot{
		ID: messages.MessageID(id), SentAt: now.Add(-time.Minute), Account: account, Country: "fr", Text: "We target the players",
	})
	s.Require().NoError(s.messages.Append(context.Background(), messages.NewRecord(message, ip, "agent")))
}

func (s *testSuite) TestAMuteHoldsTheAccountAndTheSlash64ItLastPostedFrom() {
	s.posted(bully, "2a01:e0a:9:9::1", "first")
	s.posted(bully, "2a01:e0a:1:2:aaaa::9", "latest")

	mute, err := s.useCase.Execute(context.Background(), mute_usecase.In{Account: bully})
	s.Require().NoError(err)

	s.Equal(mutes.NewMute(mutes.MuteID{15: 1}, mutes.CallerOf(bully, "2a01:e0a:1:2::/64"), mutes.Chat, now, time.Hour), mute,
		"one hour when no duration is asked")
	s.Equal([]mutes.Mute{mute}, s.mutes.Kept())
}

func (s *testSuite) TestAMuteLastsWhatWasAsked() {
	mute, err := s.useCase.Execute(context.Background(), mute_usecase.In{Account: bully, Duration: 90 * time.Minute})
	s.Require().NoError(err)

	s.Equal(now.Add(90*time.Minute), mute.Until())
}

func (s *testSuite) TestAMuteOfTheGifsHoldsTheNetworkToo() {
	s.posted(bully, "2a01:e0a:1:2:aaaa::9", "latest")

	mute, err := s.useCase.Execute(context.Background(), mute_usecase.In{Account: bully, Kind: mutes.Gifs})
	s.Require().NoError(err)

	s.Equal(mutes.NewMute(mutes.MuteID{15: 1}, mutes.CallerOf(bully, "2a01:e0a:1:2::/64"), mutes.Gifs, now, time.Hour), mute)
	s.Equal([]mutes.Mute{mute}, s.mutes.Kept())
}

func (s *testSuite) TestAMuteIsRecordedWithItsReasonAndWhoGaveIt() {
	s.posted(bully, "2a01:e0a:1:2:aaaa::9", "latest")

	_, err := s.useCase.Execute(context.Background(), mute_usecase.In{
		Account: bully, Kind: mutes.Gifs, Duration: 15 * time.Minute, Reason: "  a gore GIF, 2nd offence ", By: claude,
	})
	s.Require().NoError(err)

	s.Equal([]moderation.Snapshot{{
		Account: bully, At: now, Kind: moderation.Muted, MuteKind: mutes.Gifs, Duration: 15 * time.Minute,
		Reason: "a gore GIF, 2nd offence", By: claude,
	}}, s.history.Kept(), "the account alone: never the network the mute also holds")
}

func (s *testSuite) TestAnInvalidReasonIsRefusedAndNothingIsKept() {
	_, err := s.useCase.Execute(context.Background(), mute_usecase.In{Account: bully, Reason: string([]byte{0xff})})

	s.Require().ErrorIs(err, moderation.ErrInvalidReason)
	s.Empty(s.mutes.Kept())
	s.Empty(s.history.Kept())
}

func (s *testSuite) TestAMuteTheHistoryCannotRecordFails() {
	s.history.FailWith(errors.New("postgres is down"))

	_, err := s.useCase.Execute(context.Background(), mute_usecase.In{Account: bully})

	s.Require().Error(err, "a mute is never kept without its record: the transaction keeps neither")
}

func (s *testSuite) TestAnAccountOnlyMuteLeavesTheNetworkItPostedFrom() {
	s.posted(bully, "2a02:8086:c95:7580::9", "latest")

	mute, err := s.useCase.Execute(context.Background(), mute_usecase.In{
		Account: bully, Duration: 30 * 24 * time.Hour, AccountOnly: true,
	})
	s.Require().NoError(err)

	s.Equal(mutes.NewMute(mutes.MuteID{15: 1}, mutes.CallerOf(bully, mutes.NoScope), mutes.Chat, now, 30*24*time.Hour), mute,
		"the household behind the /64 keeps the chat")
	s.Equal([]mutes.Mute{mute}, s.mutes.Kept())
}

func (s *testSuite) TestAnAccountThatNeverPostedIsMutedAlone() {
	mute, err := s.useCase.Execute(context.Background(), mute_usecase.In{Account: quiet})
	s.Require().NoError(err)

	s.Equal(mutes.CallerOf(quiet, mutes.NoScope), mute.Caller())
}

func (s *testSuite) TestNoAccountIsRefusedAndNothingIsKept() {
	_, err := s.useCase.Execute(context.Background(), mute_usecase.In{Account: messages.NoAccount})

	s.Require().ErrorIs(err, mutes.ErrNoAccount)
	s.Empty(s.mutes.Kept())
	s.Empty(s.history.Kept())
}

func (s *testSuite) TestAnInvalidDurationIsRefusedAndNothingIsKept() {
	_, err := s.useCase.Execute(context.Background(), mute_usecase.In{Account: bully, Duration: -time.Hour})

	s.Require().ErrorIs(err, mutes.ErrInvalidDuration)
	s.Empty(s.mutes.Kept())
}

func (s *testSuite) TestAMuteWithNoIDIsNotKept() {
	_, err := s.build(s.mutes, failingIDs{}).Execute(context.Background(), mute_usecase.In{Account: bully})

	s.Require().Error(err)
	s.Empty(s.mutes.Kept())
}

func (s *testSuite) TestAMuteThatCannotBeKeptFails() {
	_, err := s.build(failingSaver{}, &mutes.SequentialIDs{}).Execute(context.Background(), mute_usecase.In{Account: bully})

	s.Require().Error(err)
}

type failingSaver struct{}

func (failingSaver) Save(context.Context, mutes.Mute) error { return errors.New("postgres is down") }

type failingIDs struct{}

func (failingIDs) NewID() (mutes.MuteID, error) { return mutes.MuteID{}, errors.New("no entropy") }
