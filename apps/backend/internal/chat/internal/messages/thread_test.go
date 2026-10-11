package messages_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/inmemory_message_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type fakeNamer struct {
	named map[messages.AccountID]messages.Author
	asked [][]messages.AccountID
	err   error
}

func (f *fakeNamer) Authors(_ context.Context, accounts []messages.AccountID) (map[messages.AccountID]messages.Author, error) {
	f.asked = append(f.asked, accounts)
	if f.err != nil {
		return nil, f.err
	}
	return f.named, nil
}

var (
	bob    = messages.AccountID{15: 2}
	adaNow = messages.AuthorOf("Ada_L", true, 7, 12, messages.Title{})
)

type threading struct {
	t       *testing.T
	storage *inmemory_message_storage.Storage
	namer   *fakeNamer
	thread  messages.Thread
	draft   messages.Message
}

func newThreading(t *testing.T, size int) threading {
	t.Helper()

	storage := inmemory_message_storage.New()
	namer := &fakeNamer{named: map[messages.AccountID]messages.Author{ada: adaNow}}
	return threading{
		t:       t,
		storage: storage,
		namer:   namer,
		thread:  messages.NewThread(storage, namer, cptime.NewFixedClock(draftedAt), messages.NewWindow(size, 24*time.Hour)),
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
	assert.Empty(t, thread.namer.asked)
}

func TestAReplyQuotesWhatAShownMessageSaidUnderWhoItsAccountIsNow(t *testing.T) {
	thread := newThreading(t, 10)
	thread.sent(messages.Snapshot{ID: "hello", SentAt: draftedAt.Add(-time.Hour), Account: ada, Country: "fr", Text: "hello planet"})

	reply, err := thread.thread.Reply(t.Context(), thread.draft, "hello")

	require.NoError(t, err)
	want := thread.draft.TakeSnapshot()
	want.Quote = messages.Quote{MessageID: "hello", Author: adaNow, Text: "hello planet"}
	assert.Equal(t, want, reply.TakeSnapshot(), "the draft, and what it answers")
	assert.Equal(t, [][]messages.AccountID{{ada}}, thread.namer.asked)
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
	assert.Empty(t, thread.namer.asked, "nobody is asked about a message that is not answered")
}

func TestAReplyToADeletedAccountNoLongerNamesIt(t *testing.T) {
	thread := newThreading(t, 10)
	thread.sent(messages.Snapshot{ID: "hello", SentAt: draftedAt.Add(-time.Hour), Account: messages.AccountID{15: 9}, Text: "hello planet"})

	quote := thread.quote("hello")

	assert.Equal(t, messages.DeletedName, quote.Author.Name())
	assert.Equal(t, "hello planet", quote.Text, "what it said stays")
}

func TestAReplyToAMessageFromBeforeAccountsKeepsTheNameItCarries(t *testing.T) {
	thread := newThreading(t, 10)
	thread.sent(messages.Snapshot{
		ID: "legacy", SentAt: draftedAt.Add(-time.Hour), Author: messages.AuthorOf("guest_Bob", false, 0, 0, messages.Title{}), Text: "hello",
	})

	quote := thread.quote("legacy")

	assert.Equal(t, "guest_Bob", quote.Author.Name())
	assert.Empty(t, thread.namer.asked, "it has no account to read a name from")
}

func TestAReplyToAReplyQuotesThatReplyAlone(t *testing.T) {
	thread := newThreading(t, 10)
	thread.sent(messages.Snapshot{ID: "question", SentAt: draftedAt.Add(-2 * time.Hour), Account: ada, Text: "who holds the north"})
	answer, err := thread.thread.Reply(t.Context(), thread.draft, "question")
	require.NoError(t, err)
	require.NoError(t, thread.storage.Append(t.Context(), messages.NewRecord(answer, "browser-1", "203.0.113.7", "test-agent")))
	thread.namer.named[bob] = messages.AuthorOf("Bob", false, 0, 0, messages.Title{})

	quote := thread.quote("draft")

	assert.Equal(t, messages.Quote{MessageID: "draft", Author: thread.namer.named[bob], Text: "we do"}, quote)
}

func TestAReplyNobodyCouldBeAskedAboutIsRefused(t *testing.T) {
	thread := newThreading(t, 10)
	thread.sent(messages.Snapshot{ID: "hello", SentAt: draftedAt.Add(-time.Hour), Account: ada, Text: "hello planet"})
	thread.namer.err = errors.New("the player module is stuck")

	_, err := thread.thread.Reply(t.Context(), thread.draft, "hello")

	require.ErrorIs(t, err, messages.ErrAuthorUnavailable)
}
