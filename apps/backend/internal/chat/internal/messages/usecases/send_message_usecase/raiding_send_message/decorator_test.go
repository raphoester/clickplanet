package raiding_send_message_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase/raiding_send_message"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids/inmemory_raid_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	now    = time.Date(2026, 11, 7, 20, 0, 0, 0, time.UTC)
	raider = messages.AccountID{15: 1}
	ada    = messages.AccountID{15: 2}
)

type testSuite struct {
	suite.Suite

	store     *inmemory_raid_storage.Storage
	clock     *cptime.FixedClock
	authors   *fakeAuthors
	inner     *recordingUseCase
	decorator *raiding_send_message.Decorator
}

func (s *testSuite) SetupTest() {
	s.store = inmemory_raid_storage.New()
	s.clock = cptime.NewFixedClock(now)
	s.authors = &fakeAuthors{guests: map[messages.AccountID]bool{raider: true}}
	s.inner = &recordingUseCase{}
	s.decorator = raiding_send_message.New(s.inner, raids.NewBook(s.store, s.clock), s.authors)
}

func (s *testSuite) raidMode(duration time.Duration) {
	mode, err := raids.Start(now, duration, "a raid of guests")
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), mode))
}

func (s *testSuite) post(account messages.AccountID) error {
	_, err := s.decorator.Execute(s.T().Context(), send_message_usecase.In{Account: account, Text: "hello"})
	return err
}

func (s *testSuite) TestWithNoRaidModeAGuestPostsAndThePlayerModuleIsNotAsked() {
	s.Require().NoError(s.post(raider))

	s.Equal(1, s.inner.calls)
	s.Empty(s.authors.asked, "the use case introduces the sender, once")
}

func (s *testSuite) TestWhileRaidModeRunsAGuestIsRefusedAndNothingIsSent() {
	s.raidMode(time.Hour)

	err := s.post(raider)

	s.Require().ErrorIs(err, raids.ErrGuest)
	s.Zero(s.inner.calls)
	s.Equal([]messages.AccountID{raider}, s.authors.asked)
}

func (s *testSuite) TestWhileRaidModeRunsASignedInPlayerPosts() {
	s.raidMode(time.Hour)

	s.Require().NoError(s.post(ada))
	s.Equal(1, s.inner.calls)
}

func (s *testSuite) TestOnceRaidModeEndedByItselfAGuestPostsAgain() {
	s.raidMode(time.Hour)
	s.clock.Advance(time.Hour)

	s.Require().NoError(s.post(raider))
	s.Empty(s.authors.asked)
}

func (s *testSuite) TestNoAccountIsLeftToTheUseCaseToRefuse() {
	s.raidMode(time.Hour)

	s.Require().NoError(s.post(messages.NoAccount))
	s.Equal(1, s.inner.calls)
	s.Empty(s.authors.asked)
}

func (s *testSuite) TestAPlayerModuleThatCannotAnswerRefusesThePostWhileRaidModeRuns() {
	s.raidMode(time.Hour)
	s.authors.err = errors.New("player is stuck")

	err := s.post(ada)

	s.Require().ErrorIs(err, messages.ErrAuthorUnavailable)
	s.Zero(s.inner.calls)
}

func (s *testSuite) TestAFailureToReadTheRaidModeRefusesThePost() {
	cause := errors.New("postgres is down")
	s.store.FailWith(cause)

	err := s.post(ada)

	s.Require().ErrorIs(err, cause)
	s.Zero(s.inner.calls)
}

type fakeAuthors struct {
	guests map[messages.AccountID]bool
	err    error
	asked  []messages.AccountID
}

func (f *fakeAuthors) Author(_ context.Context, account messages.AccountID) (messages.Author, error) {
	f.asked = append(f.asked, account)
	if f.err != nil {
		return messages.Author{}, f.err
	}
	if f.guests[account] {
		return messages.GuestOf("guest_0b1c2d"), nil
	}
	return messages.AuthorOf("Ada_L"), nil
}

type recordingUseCase struct{ calls int }

func (r *recordingUseCase) Execute(context.Context, send_message_usecase.In) (messages.Message, error) {
	r.calls++
	return messages.Message{}, nil
}
