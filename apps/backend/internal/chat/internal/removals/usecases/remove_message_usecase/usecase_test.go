package remove_message_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation/inmemory_moderation_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/usecases/remove_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	now       = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	retention = 24 * time.Hour
	bob       = messages.AccountID{15: 2}
	claude    = moderation.OperatorOf(messages.AccountID{15: 9}, "Claude")
)

type keptMessages struct {
	kept  map[messages.MessageID]messages.Message
	since time.Time
	err   error
}

func (k *keptMessages) KeptMessage(_ context.Context, id messages.MessageID, since time.Time) (messages.Message, error) {
	k.since = since
	if k.err != nil {
		return messages.Message{}, k.err
	}
	message, kept := k.kept[id]
	if !kept {
		return messages.Message{}, messages.ErrNotKept
	}
	return message, nil
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
	history  *inmemory_moderation_storage.Storage
	feed     *recordedFeed
}

func (s *testSuite) SetupTest() {
	s.messages = &keptMessages{kept: map[messages.MessageID]messages.Message{
		"insult": messages.Restored(s.T(), messages.Snapshot{
			ID: "insult", SentAt: now.Add(-time.Hour), Account: bob, Country: "fr", Text: "you all suck",
		}),
	}}
	s.removals = &recordedRemovals{}
	s.history = inmemory_moderation_storage.New()
	s.feed = &recordedFeed{}
}

func (s *testSuite) remove(id messages.MessageID) (time.Time, error) {
	return s.removeWith(remove_message_usecase.In{Message: id, Reason: "an insult aimed at a player", By: claude})
}

func (s *testSuite) removeWith(in remove_message_usecase.In) (time.Time, error) {
	return remove_message_usecase.New(cpatomic.FakeTransactioner{}, s.messages, s.removals, s.history, s.feed,
		messages.NewWindow(200, retention), cptime.NewFixedClock(now)).Execute(s.T().Context(), in)
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

func (s *testSuite) TestARemovalIsRecordedOnItsAuthorWithACopyOfWhatItSaid() {
	_, err := s.remove("insult")
	s.Require().NoError(err)

	s.Equal([]moderation.Snapshot{{
		Account: bob, At: now, Kind: moderation.Removed, Message: "insult", Text: "you all suck",
		Reason: "an insult aimed at a player", By: claude,
	}}, s.history.Kept())
}

func (s *testSuite) TestARemovedGifIsRecordedAsItsAddress() {
	gif := messages.Gif{URL: "https://static.klipy.com/ii/935d7ab9/14/af/SE72470w.webp", Width: 220, Height: 124}
	s.messages.kept["gore"] = messages.Restored(s.T(), messages.Snapshot{
		ID: "gore", SentAt: now.Add(-time.Hour), Account: bob, Country: "fr", Gif: gif,
	})

	_, err := s.remove("gore")
	s.Require().NoError(err)

	s.Require().Len(s.history.Kept(), 1)
	s.Equal(gif, s.history.Kept()[0].Gif)
	s.Empty(s.history.Kept()[0].Text)
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
	s.Empty(s.history.Kept())
	s.Empty(s.feed.updates)
}

func (s *testSuite) TestAnInvalidReasonRemovesNothing() {
	_, err := s.removeWith(remove_message_usecase.In{Message: "insult", Reason: string([]byte{0xff})})

	s.Require().ErrorIs(err, moderation.ErrInvalidReason)
	s.Empty(s.removals.saved)
	s.Empty(s.history.Kept())
	s.Empty(s.feed.updates)
}

func (s *testSuite) TestARemovalThatCannotBeKeptIsNotTold() {
	s.removals.err = errors.New("postgres is down")

	_, err := s.remove("insult")

	s.Require().Error(err)
	s.Empty(s.feed.updates)
}

func (s *testSuite) TestARemovalTheHistoryCannotRecordIsNotTold() {
	s.history.FailWith(errors.New("postgres is down"))

	_, err := s.remove("insult")

	s.Require().Error(err, "the transaction keeps neither")
	s.Empty(s.feed.updates)
}

func (s *testSuite) TestAFailedReadRemovesNothing() {
	s.messages.err = errors.New("postgres is down")

	_, err := s.remove("insult")

	s.Require().Error(err)
	s.Require().NotErrorIs(err, removals.ErrUnknownMessage)
	s.Empty(s.removals.saved)
}
