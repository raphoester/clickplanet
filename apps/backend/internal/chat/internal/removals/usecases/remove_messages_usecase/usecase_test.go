package remove_messages_usecase_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation/inmemory_moderation_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/usecases/remove_messages_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	now       = time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	retention = 24 * time.Hour
	bob       = messages.AccountID{15: 2}
	eve       = messages.AccountID{15: 3}
	claude    = moderation.OperatorOf(messages.AccountID{15: 9}, "Claude")
)

type keptMessages struct {
	kept  map[messages.MessageID]messages.Message
	asked []messages.MessageID
	since time.Time
	err   error
}

func (k *keptMessages) KeptMessage(_ context.Context, id messages.MessageID, since time.Time) (messages.Message, error) {
	k.asked = append(k.asked, id)
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
	saved  []removals.Snapshot
	failOn messages.MessageID
}

func (r *recordedRemovals) Save(_ context.Context, removal removals.Snapshot) error {
	if removal.MessageID == r.failOn {
		return errors.New("postgres is down")
	}
	r.saved = append(r.saved, removal)
	return nil
}

type recordedFeed struct {
	updates []feed.Update
}

func (r *recordedFeed) Publish(update feed.Update) { r.updates = append(r.updates, update) }

func (r *recordedFeed) removed() []messages.MessageID {
	var out []messages.MessageID
	for _, update := range r.updates {
		if id, removed := update.Removed(); removed {
			out = append(out, id)
		}
	}
	return out
}

type testSuite struct {
	suite.Suite

	messages *keptMessages
	removals *recordedRemovals
	history  *inmemory_moderation_storage.Storage
	feed     *recordedFeed
}

func (s *testSuite) SetupTest() {
	s.messages = &keptMessages{kept: map[messages.MessageID]messages.Message{}}
	s.keep(messages.Snapshot{ID: "insult", SentAt: now.Add(-time.Hour), Account: bob, Country: "fr", Text: "you all suck"})
	s.keep(messages.Snapshot{ID: "spam", SentAt: now.Add(-time.Minute), Account: eve, Country: "de", Text: "join my server"})
	s.removals = &recordedRemovals{}
	s.history = inmemory_moderation_storage.New()
	s.feed = &recordedFeed{}
}

func (s *testSuite) keep(message messages.Snapshot) {
	s.messages.kept[message.ID] = messages.Restored(s.T(), message)
}

func (s *testSuite) remove(removals ...remove_messages_usecase.Removal) (time.Time, error) {
	return remove_messages_usecase.New(cpatomic.FakeTransactioner{}, s.messages, s.removals, s.history, s.feed,
		messages.NewWindow(200, retention), cptime.NewFixedClock(now)).Execute(s.T().Context(),
		remove_messages_usecase.In{Removals: removals, By: claude})
}

func (s *testSuite) assertNothingDone() {
	s.Empty(s.removals.saved)
	s.Empty(s.history.Kept())
	s.Empty(s.feed.updates)
}

func (s *testSuite) TestEveryMessageIsKeptThenToldToEveryStream() {
	removedAt, err := s.remove(
		remove_messages_usecase.Removal{Message: "insult", Reason: "offence 1 (chat): an insult aimed at a player"},
		remove_messages_usecase.Removal{Message: "spam", Reason: "spam, no offence"},
	)
	s.Require().NoError(err)

	s.Equal(now, removedAt)
	s.Equal([]removals.Snapshot{{MessageID: "insult", RemovedAt: now}, {MessageID: "spam", RemovedAt: now}},
		s.removals.saved)
	s.Equal([]messages.MessageID{"insult", "spam"}, s.feed.removed())
}

func (s *testSuite) TestEachRemovalIsRecordedOnItsAuthorWithItsOwnReason() {
	_, err := s.remove(
		remove_messages_usecase.Removal{Message: "insult", Reason: "offence 1 (chat): an insult aimed at a player"},
		remove_messages_usecase.Removal{Message: "spam", Reason: "spam, no offence"},
	)
	s.Require().NoError(err)

	s.Equal([]moderation.Snapshot{
		{
			Account: bob, At: now, Kind: moderation.Removed, Message: "insult", Text: "you all suck",
			Reason: "offence 1 (chat): an insult aimed at a player", By: claude,
		},
		{
			Account: eve, At: now, Kind: moderation.Removed, Message: "spam", Text: "join my server",
			Reason: "spam, no offence", By: claude,
		},
	}, s.history.Kept())
}

func (s *testSuite) TestOnlyMessagesInsideTheRetentionCanBeRemoved() {
	_, err := s.remove(remove_messages_usecase.Removal{Message: "insult"})
	s.Require().NoError(err)

	s.Equal(now.Add(-retention), s.messages.since)
}

func (s *testSuite) TestOneMessageNobodyKeptRemovesNoneAndIsNamed() {
	_, err := s.remove(
		remove_messages_usecase.Removal{Message: "insult"},
		remove_messages_usecase.Removal{Message: "pruned"},
		remove_messages_usecase.Removal{Message: "spam"},
	)

	s.Require().ErrorIs(err, removals.ErrUnknownMessage)
	s.Contains(err.Error(), `"pruned"`)
	s.assertNothingDone()
}

func (s *testSuite) TestTheSameMessageTwiceIsRefusedBeforeAnyRead() {
	_, err := s.remove(
		remove_messages_usecase.Removal{Message: "insult", Reason: "offence 1 (chat): an insult"},
		remove_messages_usecase.Removal{Message: "insult", Reason: "offence 2 (chat): an insult"},
	)

	s.Require().ErrorIs(err, remove_messages_usecase.ErrRemovedTwice)
	s.Contains(err.Error(), `"insult"`)
	s.Empty(s.messages.asked)
	s.assertNothingDone()
}

func (s *testSuite) TestOneInvalidReasonRemovesNoneAndIsNamed() {
	_, err := s.remove(
		remove_messages_usecase.Removal{Message: "insult", Reason: "an insult"},
		remove_messages_usecase.Removal{Message: "spam", Reason: string([]byte{0xff})},
	)

	s.Require().ErrorIs(err, moderation.ErrInvalidReason)
	s.Contains(err.Error(), `"spam"`)
	s.Empty(s.messages.asked)
	s.assertNothingDone()
}

func (s *testSuite) TestNoRemovalIsRefused() {
	_, err := s.remove()

	s.Require().ErrorIs(err, remove_messages_usecase.ErrNoRemoval)
	s.assertNothingDone()
}

func (s *testSuite) TestAPageOfMessagesIsRemovedAtOnceAndNoMore() {
	page := make([]remove_messages_usecase.Removal, 0, remove_messages_usecase.MaxRemovals+1)
	for i := range remove_messages_usecase.MaxRemovals + 1 {
		id := messages.MessageID(fmt.Sprintf("spam-%d", i))
		s.keep(messages.Snapshot{ID: id, SentAt: now.Add(-time.Minute), Account: eve, Country: "de", Text: "spam"})
		page = append(page, remove_messages_usecase.Removal{Message: id, Reason: "spam, no offence"})
	}

	_, err := s.remove(page...)
	s.Require().ErrorIs(err, remove_messages_usecase.ErrTooManyRemovals)
	s.assertNothingDone()

	_, err = s.remove(page[:remove_messages_usecase.MaxRemovals]...)
	s.Require().NoError(err)
	s.Len(s.removals.saved, remove_messages_usecase.MaxRemovals)
	s.Len(s.history.Kept(), remove_messages_usecase.MaxRemovals)
	s.Len(s.feed.removed(), remove_messages_usecase.MaxRemovals)
}

func (s *testSuite) TestAMessageAlreadyRemovedKeepsItsFirstAction() {
	_, err := s.remove(remove_messages_usecase.Removal{Message: "insult", Reason: "offence 1 (chat): an insult"})
	s.Require().NoError(err)

	_, err = s.remove(
		remove_messages_usecase.Removal{Message: "insult", Reason: "offence 1 (chat): an insult"},
		remove_messages_usecase.Removal{Message: "spam", Reason: "spam, no offence"},
	)
	s.Require().NoError(err, "a run tried again removes what is left")

	s.Len(s.history.Kept(), 2, "the insult counts once")
}

func (s *testSuite) TestARemovalThatCannotBeKeptTellsNone() {
	s.removals.failOn = "spam"

	_, err := s.remove(remove_messages_usecase.Removal{Message: "insult"}, remove_messages_usecase.Removal{Message: "spam"})

	s.Require().Error(err)
	s.Contains(err.Error(), `"spam"`)
	s.Empty(s.feed.updates, "the transaction keeps none, so no stream hears of the first")
}

func (s *testSuite) TestARemovalTheHistoryCannotRecordTellsNone() {
	s.history.FailWith(errors.New("postgres is down"))

	_, err := s.remove(remove_messages_usecase.Removal{Message: "insult"}, remove_messages_usecase.Removal{Message: "spam"})

	s.Require().Error(err, "the transaction keeps neither")
	s.Empty(s.feed.updates)
}

func (s *testSuite) TestAFailedReadRemovesNothing() {
	s.messages.err = errors.New("postgres is down")

	_, err := s.remove(remove_messages_usecase.Removal{Message: "insult"})

	s.Require().Error(err)
	s.Require().NotErrorIs(err, removals.ErrUnknownMessage)
	s.assertNothingDone()
}
