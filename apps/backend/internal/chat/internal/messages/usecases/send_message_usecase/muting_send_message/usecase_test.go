package muting_send_message_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase/muting_send_message"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/inmemory_mute_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	mutedAt  = time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC)
	bully    = messages.AccountID{15: 1}
	newGuest = messages.AccountID{15: 2}
)

type testSuite struct {
	suite.Suite

	store     *inmemory_mute_storage.Storage
	clock     *cptime.FixedClock
	inner     *recordingUseCase
	decorator *muting_send_message.Decorator
}

func (s *testSuite) SetupTest() {
	store := inmemory_mute_storage.New()
	s.store = store
	s.Require().NoError(store.Save(context.Background(),
		mutes.NewMute(mutes.MuteID{1}, mutes.NewCaller(bully, "2a01:e0a:1:2::1"), mutes.Chat, mutedAt, time.Hour)))
	s.Require().NoError(store.Save(context.Background(),
		mutes.NewMute(mutes.MuteID{2}, mutes.NewCaller(messages.AccountID{15: 3}, "203.0.113.7"), mutes.Chat, mutedAt, time.Hour)))

	s.clock = cptime.NewFixedClock(mutedAt.Add(time.Minute))
	s.inner = &recordingUseCase{}
	s.decorator = muting_send_message.New(s.inner, mutes.NewBook(store, s.clock))
}

func (s *testSuite) post(account messages.AccountID, ip string) error {
	_, err := s.decorator.Execute(cpctx.AddIPToContext(context.Background(), ip),
		send_message_usecase.In{Account: account, CountryID: "fr", Text: "hello"})
	return err
}

func (s *testSuite) postGif(account messages.AccountID, ip string) error {
	_, err := s.decorator.Execute(cpctx.AddIPToContext(context.Background(), ip), send_message_usecase.In{
		Account: account, CountryID: "fr",
		Gif: messages.Gif{URL: "https://static.klipy.com/ii/935d7ab9/14/af/SE72470w.webp", Width: 220, Height: 124},
	})
	return err
}

func (s *testSuite) mutedFromGifs(id byte, account messages.AccountID, ip string, duration time.Duration) {
	s.Require().NoError(s.store.Save(context.Background(),
		mutes.NewMute(mutes.MuteID{id}, mutes.NewCaller(account, ip), mutes.Gifs, mutedAt, duration)))
}

func (s *testSuite) TestAMuteOfTheGifsRefusesAGifAndSaysSo() {
	s.mutedFromGifs(3, newGuest, "198.51.100.1", time.Hour)

	err := s.postGif(newGuest, "198.51.100.1")

	var refusal mutes.Refusal
	s.Require().ErrorAs(err, &refusal)
	s.Equal(mutes.Gifs, refusal.Kind())
	s.Zero(s.inner.calls)
}

func (s *testSuite) TestAMuteOfTheGifsLetsTheTextGo() {
	s.mutedFromGifs(3, newGuest, "198.51.100.1", time.Hour)

	s.Require().NoError(s.post(newGuest, "198.51.100.1"))
	s.Equal(1, s.inner.calls)
}

func (s *testSuite) TestAPlayerMutedBothWaysIsToldTheMuteOfTheChat() {
	s.mutedFromGifs(3, bully, "2a01:e0a:1:2::1", 24*time.Hour)

	err := s.postGif(bully, "2a01:e0a:1:2::1")

	var refusal mutes.Refusal
	s.Require().ErrorAs(err, &refusal)
	s.Equal(mutes.Chat, refusal.Kind())
	s.Equal(mutedAt.Add(time.Hour), refusal.Until())
}

func (s *testSuite) TestAMutedAccountCannotPostFromAnyNetwork() {
	err := s.post(bully, "2a01:e0a:77:1::1")

	var refusal mutes.Refusal
	s.Require().ErrorAs(err, &refusal)
	s.Equal(mutedAt.Add(time.Hour), refusal.Until())
	s.Zero(s.inner.calls, "a muted message is never named, kept nor published")
}

func (s *testSuite) TestAFreshGuestOnAMutedSlash64CannotPost() {
	err := s.post(newGuest, "2a01:e0a:1:2:dead:beef:0:5")

	s.Require().ErrorIs(err, mutes.ErrMuted)
	s.Zero(s.inner.calls)
}

func (s *testSuite) TestAGuestOnTheNextSlash64Posts() {
	s.Require().NoError(s.post(newGuest, "2a01:e0a:1:3::1"))

	s.Equal(1, s.inner.calls)
}

func (s *testSuite) TestTheNextIPv4AddressPosts() {
	s.Require().ErrorIs(s.post(newGuest, "203.0.113.7"), mutes.ErrMuted)
	s.Require().NoError(s.post(newGuest, "203.0.113.8"))
}

func (s *testSuite) TestAMuteThatEndedLetsThePlayerPostAgain() {
	s.clock.Advance(time.Hour)

	s.Require().NoError(s.post(bully, "2a01:e0a:1:2::1"))
	s.Equal(1, s.inner.calls)
}

type recordingUseCase struct{ calls int }

func (r *recordingUseCase) Execute(context.Context, send_message_usecase.In) (messages.Message, error) {
	r.calls++
	return messages.Message{}, nil
}
