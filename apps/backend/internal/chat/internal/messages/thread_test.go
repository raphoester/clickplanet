package messages_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/inmemory_message_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var bob = messages.AccountID{15: 2}

type threading struct {
	t       *testing.T
	storage *inmemory_message_storage.Storage
	thread  messages.Thread
	draft   messages.Message
}

func newThreading(t *testing.T, size int) threading {
	t.Helper()

	storage := inmemory_message_storage.New()
	return threading{
		t:       t,
		storage: storage,
		thread:  messages.NewThread(storage, cptime.NewFixedClock(draftedAt), messages.NewWindow(size, 24*time.Hour)),
		draft:   messages.Restored(t, messages.Snapshot{ID: "draft", SentAt: draftedAt, Account: bob, Country: "de", Text: "we do"}),
	}
}

func (s threading) sent(kept messages.Snapshot) {
	s.t.Helper()
	require.NoError(s.t, s.storage.Append(s.t.Context(),
		messages.NewRecord(messages.Restored(s.t, kept), "browser-1", "203.0.113.7", "test-agent")))
}

func (s threading) quote(replyTo messages.MessageID) messages.Quote {
	s.t.Helper()
	reply, err := s.thread.Reply(s.t.Context(), s.draft, replyTo)
	require.NoError(s.t, err)
	return reply.TakeSnapshot().Quote
}

func TestADraftThatAnswersNothingIsLeftAsItIs(t *testing.T) {
	thread := newThreading(t, 10)

	reply, err := thread.thread.Reply(t.Context(), thread.draft, "")

	require.NoError(t, err)
	assert.Equal(t, thread.draft, reply)
}

func TestAReplyQuotesWhatAShownMessageSaidAndWhoseItIs(t *testing.T) {
	thread := newThreading(t, 10)
	thread.sent(messages.Snapshot{ID: "hello", SentAt: draftedAt.Add(-time.Hour), Account: ada, Country: "fr", Text: "hello planet"})

	reply, err := thread.thread.Reply(t.Context(), thread.draft, "hello")

	require.NoError(t, err)
	want := thread.draft.TakeSnapshot()
	want.Quote = messages.Quote{MessageID: "hello", Account: ada, Text: "hello planet"}
	assert.Equal(t, want, reply.TakeSnapshot(), "the draft, and what it answers")
}

func TestAMessageThatIsNotShownCannotBeRepliedTo(t *testing.T) {
	thread := newThreading(t, 1)
	thread.sent(messages.Snapshot{ID: "pushed-out", SentAt: draftedAt.Add(-2 * time.Hour), Account: ada, Text: "old"})
	thread.sent(messages.Snapshot{ID: "expired", SentAt: draftedAt.Add(-25 * time.Hour), Account: ada, Text: "older"})
	thread.sent(messages.Snapshot{ID: "shown", SentAt: draftedAt.Add(-time.Hour), Account: ada, Text: "new"})

	for _, id := range []messages.MessageID{"pushed-out", "expired", "never-sent"} {
		_, err := thread.thread.Reply(t.Context(), thread.draft, id)

		require.ErrorIs(t, err, messages.ErrNotShown, id)
	}
}

func TestAReplyToAReplyQuotesThatReplyAlone(t *testing.T) {
	thread := newThreading(t, 10)
	thread.sent(messages.Snapshot{ID: "question", SentAt: draftedAt.Add(-2 * time.Hour), Account: ada, Text: "who holds the north"})
	answer, err := thread.thread.Reply(t.Context(), thread.draft, "question")
	require.NoError(t, err)
	require.NoError(t, thread.storage.Append(t.Context(), messages.NewRecord(answer, "browser-1", "203.0.113.7", "test-agent")))

	quote := thread.quote("draft")

	assert.Equal(t, messages.Quote{MessageID: "draft", Account: bob, Text: "we do"}, quote)
}

func TestAMessageOfAnotherRoomCannotBeRepliedTo(t *testing.T) {
	thread := newThreading(t, 10)
	thread.sent(messages.Snapshot{
		ID: "plan", SentAt: draftedAt.Add(-time.Hour), Account: ada, Room: messages.FlagRoom("fr"), Text: "north at 8",
	})

	_, err := thread.thread.Reply(t.Context(), thread.draft, "plan")

	require.ErrorIs(t, err, messages.ErrNotShown, "the draft is in the chat everyone reads")
}
