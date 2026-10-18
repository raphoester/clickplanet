package remove_message_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/usecases/remove_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	now       = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	retention = 24 * time.Hour
)

type keptMessages struct {
	ids   map[messages.MessageID]bool
	since time.Time
	err   error
}

func (k *keptMessages) Kept(_ context.Context, id messages.MessageID, since time.Time) (bool, error) {
	k.since = since
	return k.ids[id], k.err
}

type recordedRemovals struct {
	saved []removals.Snapshot
	err   error
}

func (r *recordedRemovals) Save(_ context.Context, removal removals.Snapshot) error {
	if r.err != nil {
		return r.err
	}
	r.saved = append(r.saved, removal)
	return nil
}

type recordedFeed struct {
	updates []feed.Update
}

func (r *recordedFeed) Publish(update feed.Update) { r.updates = append(r.updates, update) }

type testSuite struct {
	suite.Suite

	messages *keptMessages
	removals *recordedRemovals
	feed     *recordedFeed
}

func (s *testSuite) SetupTest() {
	s.messages = &keptMessages{ids: map[messages.MessageID]bool{"insult": true}}
	s.removals = &recordedRemovals{}
	s.feed = &recordedFeed{}
}

func (s *testSuite) remove(id messages.MessageID) (time.Time, error) {
	return remove_message_usecase.New(s.messages, s.removals, s.feed,
		messages.NewWindow(200, retention), cptime.NewFixedClock(now)).Execute(s.T().Context(), id)
}

func (s *testSuite) TestARemovalIsKeptThenToldToEveryStream() {
	removedAt, err := s.remove("insult")
	s.Require().NoError(err)

	s.Equal(now, removedAt)
	s.Equal([]removals.Snapshot{{MessageID: "insult", RemovedAt: now}}, s.removals.saved)
	s.Require().Len(s.feed.updates, 1)
	removed, gone := s.feed.updates[0].Removed()
	s.True(gone)
	s.Equal(messages.MessageID("insult"), removed)
}

func (s *testSuite) TestOnlyAMessageInsideTheRetentionCanBeRemoved() {
	_, err := s.remove("insult")
	s.Require().NoError(err)

	s.Equal(now.Add(-retention), s.messages.since)
}

func (s *testSuite) TestAMessageNobodyKeptIsRefusedAndNothingIsTold() {
	_, err := s.remove("pruned")

	s.Require().ErrorIs(err, removals.ErrUnknownMessage)
	s.Empty(s.removals.saved)
	s.Empty(s.feed.updates)
}

func (s *testSuite) TestARemovalThatCannotBeKeptIsNotTold() {
	s.removals.err = errors.New("postgres is down")

	_, err := s.remove("insult")

	s.Require().Error(err)
	s.Empty(s.feed.updates)
}

func (s *testSuite) TestAFailedReadRemovesNothing() {
	s.messages.err = errors.New("postgres is down")

	_, err := s.remove("insult")

	s.Require().Error(err)
	s.Require().NotErrorIs(err, removals.ErrUnknownMessage)
	s.Empty(s.removals.saved)
}
