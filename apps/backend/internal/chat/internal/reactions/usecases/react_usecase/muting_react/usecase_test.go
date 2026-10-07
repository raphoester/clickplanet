package muting_react_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/inmemory_mute_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/usecases/react_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/usecases/react_usecase/muting_react"
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
	decorator *muting_react.Decorator
}

func (s *testSuite) SetupTest() {
	store := inmemory_mute_storage.New()
	s.Require().NoError(store.Save(context.Background(),
		mutes.NewMute(mutes.MuteID{1}, mutes.NewCaller(bully, "2a01:e0a:1:2::1"), mutedAt, time.Hour)))

	s.clock = cptime.NewFixedClock(mutedAt.Add(time.Minute))
	s.inner = &recordingUseCase{}
	s.decorator = muting_react.New(s.inner, mutes.NewBook(store, s.clock))
}

func (s *testSuite) react(account messages.AccountID, ip string) error {
	_, err := s.decorator.Execute(cpctx.AddIPToContext(context.Background(), ip),
		react_usecase.In{Account: account, MessageID: "m1", Reaction: reactions.Reaction(3), On: true})
	return err
}

func (s *testSuite) TestAMutedAccountCannotReactFromAnyNetwork() {
	err := s.react(bully, "198.51.100.1")

	var refusal mutes.Refusal
	s.Require().ErrorAs(err, &refusal)
	s.Equal(mutedAt.Add(time.Hour), refusal.Until())
	s.Zero(s.inner.calls)
}

func (s *testSuite) TestAFreshGuestOnAMutedSlash64CannotReact() {
	s.Require().ErrorIs(s.react(newGuest, "2a01:e0a:1:2::77"), mutes.ErrMuted)
	s.Zero(s.inner.calls)
}

func (s *testSuite) TestAGuestOnTheNextSlash64Reacts() {
	s.Require().NoError(s.react(newGuest, "2a01:e0a:1:3::77"))
	s.Equal(1, s.inner.calls)
}

func (s *testSuite) TestAMuteThatEndedLetsThePlayerReactAgain() {
	s.clock.Advance(time.Hour)

	s.Require().NoError(s.react(bully, "2a01:e0a:1:2::1"))
	s.Equal(1, s.inner.calls)
}

type recordingUseCase struct{ calls int }

func (r *recordingUseCase) Execute(context.Context, react_usecase.In) (react_usecase.Out, error) {
	r.calls++
	return react_usecase.Out{}, nil
}
