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

	clock     *cptime.FixedClock
	inner     *recordingUseCase
	decorator *muting_send_message.Decorator
}

func (s *testSuite) SetupTest() {
	store := inmemory_mute_storage.New()
	s.Require().NoError(store.Save(context.Background(),
		mutes.NewMute(mutes.MuteID{1}, mutes.NewCaller(bully, "2a01:e0a:1:2::1"), mutedAt, time.Hour)))
	s.Require().NoError(store.Save(context.Background(),
		mutes.NewMute(mutes.MuteID{2}, mutes.NewCaller(messages.AccountID{15: 3}, "203.0.113.7"), mutedAt, time.Hour)))

	s.clock = cptime.NewFixedClock(mutedAt.Add(time.Minute))
	s.inner = &recordingUseCase{}
	s.decorator = muting_send_message.New(s.inner, mutes.NewBook(store, s.clock))
}

func (s *testSuite) post(account messages.AccountID, ip string) error {
	_, err := s.decorator.Execute(cpctx.AddIPToContext(context.Background(), ip),
		send_message_usecase.In{Account: account, CountryID: "fr", Text: "hello"})
	return err
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
