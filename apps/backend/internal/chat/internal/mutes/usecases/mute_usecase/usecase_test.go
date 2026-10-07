package mute_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/inmemory_announcement_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/inmemory_message_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/inmemory_mute_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/mute_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	now   = time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC)
	bully = messages.AccountID{15: 1}
	quiet = messages.AccountID{15: 2}
)

type testSuite struct {
	suite.Suite

	mutes         *inmemory_mute_storage.Storage
	messages      *inmemory_message_storage.Storage
	announcements *inmemory_announcement_storage.Storage
	feed          *recordedFeed
	authors       *fakeAuthors
	useCase       *mute_usecase.UseCase
}

func (s *testSuite) SetupTest() {
	s.mutes = inmemory_mute_storage.New()
	s.messages = inmemory_message_storage.New()
	s.announcements = inmemory_announcement_storage.New()
	s.feed = &recordedFeed{}
	s.authors = &fakeAuthors{names: map[messages.AccountID]string{bully: "guest_a1b2c3", quiet: "Ada_L"}}
	s.useCase = s.build(s.mutes)
}

func (s *testSuite) build(saver mute_usecase.Saver) *mute_usecase.UseCase {
	return mute_usecase.New(saver, s.messages, s.authors,
		announce_usecase.New(s.announcements, s.feed), cptime.NewFixedClock(now))
}

func (s *testSuite) posted(account messages.AccountID, ip string, id string) {
	message := messages.NewMessage(messages.MessageID(id), now.Add(-time.Minute), account, "fr", "We target the players")
	s.Require().NoError(s.messages.Append(context.Background(), messages.NewRecord(message, "browser", ip, "agent")))
}

func (s *testSuite) TestAMuteHoldsTheAccountAndTheSlash64ItLastPostedFrom() {
	s.posted(bully, "2a01:e0a:9:9::1", "first")
	s.posted(bully, "2a01:e0a:1:2:aaaa::9", "latest")

	out, err := s.useCase.Execute(context.Background(), mute_usecase.In{Account: bully})
	s.Require().NoError(err)

	s.Equal("guest_a1b2c3", out.Name)
	s.Equal(mutes.CallerOf(bully, "2a01:e0a:1:2::/64"), out.Mute.Caller())
	s.Equal(now.Add(time.Hour), out.Mute.Until(), "one hour when no duration is asked")
	kept := s.mutes.Kept()
	s.Require().Len(kept, 1)
	s.Equal(out.Mute, kept[0])
}

func (s *testSuite) TestAMuteIsAnnouncedWithTheNameAndTheDuration() {
	s.posted(bully, "203.0.113.7", "latest")

	_, err := s.useCase.Execute(context.Background(), mute_usecase.In{Account: bully, Duration: 90 * time.Minute})
	s.Require().NoError(err)

	kept := s.announcements.Kept()
	s.Require().Len(kept, 1)
	s.Equal(announcements.KindMute, kept[0].Kind())
	s.Equal(now, kept[0].At())
	s.JSONEq(`{"name":"guest_a1b2c3","seconds":5400}`, string(kept[0].Payload()))
	s.Require().Len(s.feed.updates, 1)
	published, announced := s.feed.updates[0].Announcement()
	s.Require().True(announced, "every open chat sees it at once")
	s.Equal(kept[0], published)
}

func (s *testSuite) TestAnAccountThatNeverPostedIsMutedAlone() {
	out, err := s.useCase.Execute(context.Background(), mute_usecase.In{Account: quiet})
	s.Require().NoError(err)

	s.Equal(mutes.CallerOf(quiet, mutes.NoScope), out.Mute.Caller())
}

func (s *testSuite) TestNoAccountIsRefusedAndNothingIsKept() {
	_, err := s.useCase.Execute(context.Background(), mute_usecase.In{Account: messages.NoAccount})

	s.Require().ErrorIs(err, mutes.ErrNoAccount)
	s.Empty(s.mutes.Kept())
	s.Empty(s.announcements.Kept())
}

func (s *testSuite) TestAnInvalidDurationIsRefusedAndNothingIsKept() {
	_, err := s.useCase.Execute(context.Background(), mute_usecase.In{Account: bully, Duration: -time.Hour})

	s.Require().ErrorIs(err, mutes.ErrInvalidDuration)
	s.Empty(s.mutes.Kept())
	s.Empty(s.announcements.Kept())
}

func (s *testSuite) TestAnAccountNobodyCanNameIsNotMuted() {
	s.authors.err = errors.New("the player module is down")

	_, err := s.useCase.Execute(context.Background(), mute_usecase.In{Account: bully})

	s.Require().Error(err)
	s.Empty(s.mutes.Kept())
	s.Empty(s.announcements.Kept())
}

func (s *testSuite) TestAMuteThatCannotBeKeptIsNotAnnounced() {
	_, err := s.build(failingSaver{}).Execute(context.Background(), mute_usecase.In{Account: bully})

	s.Require().Error(err)
	s.Empty(s.announcements.Kept())
	s.Empty(s.feed.updates)
}

type recordedFeed struct{ updates []feed.Update }

func (r *recordedFeed) Publish(update feed.Update) { r.updates = append(r.updates, update) }

type fakeAuthors struct {
	names map[messages.AccountID]string
	err   error
}

func (f *fakeAuthors) Author(_ context.Context, account messages.AccountID) (messages.Author, error) {
	if f.err != nil {
		return messages.Author{}, f.err
	}
	return messages.AuthorOf(f.names[account], false, 0, 0, messages.Title{}), nil
}

type failingSaver struct{}

func (failingSaver) Save(context.Context, mutes.Mute) error { return errors.New("postgres is down") }
