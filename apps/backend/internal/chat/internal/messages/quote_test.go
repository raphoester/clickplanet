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

const teal = 7

type quoting struct {
	storage *inmemory_message_storage.Storage
	namer   *fakeNamer
	quotes  messages.Quotes
}

func newQuoting(t *testing.T, size int) quoting {
	t.Helper()

	storage := inmemory_message_storage.New()
	namer := &fakeNamer{named: map[messages.AccountID]messages.Author{
		ada: messages.AuthorOf("Ada_L", true, teal, 12, messages.Title{}),
	}}
	return quoting{
		storage: storage,
		namer:   namer,
		quotes:  messages.NewQuotes(storage, namer, cptime.NewFixedClock(draftedAt), messages.NewWindow(size, 24*time.Hour)),
	}
}

func (q quoting) sent(t *testing.T, message messages.Message) {
	t.Helper()
	require.NoError(t, q.storage.Append(t.Context(), messages.NewRecord(message, "browser-1", "203.0.113.7", "test-agent")))
}

func TestAMessageThatRepliesToNothingQuotesNothing(t *testing.T) {
	quoting := newQuoting(t, 10)

	quote, err := quoting.quotes.Quote(t.Context(), "")

	require.NoError(t, err)
	assert.True(t, quote.Empty())
	assert.Empty(t, quoting.namer.asked)
}

func TestAQuoteIsWhatAShownMessageSaidUnderWhoItsAccountIsNow(t *testing.T) {
	quoting := newQuoting(t, 10)
	quoting.sent(t, messages.NewMessage("hello", draftedAt.Add(-time.Hour), ada, "fr", "hello planet"))

	quote, err := quoting.quotes.Quote(t.Context(), "hello")

	require.NoError(t, err)
	assert.Equal(t, messages.MessageID("hello"), quote.ID())
	assert.Equal(t, "hello planet", quote.Text())
	assert.Equal(t, "Ada_L", quote.Author().Name())
	assert.Equal(t, int32(teal), quote.Author().Color())
	assert.Equal(t, [][]messages.AccountID{{ada}}, quoting.namer.asked)
}

func TestAMessageThatIsNotShownCannotBeRepliedTo(t *testing.T) {
	quoting := newQuoting(t, 1)
	quoting.sent(t, messages.NewMessage("pushed-out", draftedAt.Add(-2*time.Hour), ada, "fr", "old"))
	quoting.sent(t, messages.NewMessage("expired", draftedAt.Add(-25*time.Hour), ada, "fr", "older"))
	quoting.sent(t, messages.NewMessage("shown", draftedAt.Add(-time.Hour), ada, "fr", "new"))

	for _, id := range []messages.MessageID{"pushed-out", "expired", "never-sent"} {
		_, err := quoting.quotes.Quote(t.Context(), id)

		require.ErrorIs(t, err, messages.ErrNotShown, id)
	}
	assert.Empty(t, quoting.namer.asked, "nobody is asked about a message that is not quoted")
}

func TestAQuoteOfADeletedAccountIsNoLongerNamed(t *testing.T) {
	quoting := newQuoting(t, 10)
	quoting.sent(t, messages.NewMessage("hello", draftedAt.Add(-time.Hour), messages.AccountID{15: 9}, "fr", "hello planet"))

	quote, err := quoting.quotes.Quote(t.Context(), "hello")

	require.NoError(t, err)
	assert.Equal(t, messages.DeletedName, quote.Author().Name())
	assert.Equal(t, "hello planet", quote.Text(), "what it said stays")
}

func TestAQuoteFromBeforeAccountsKeepsTheNameItsMessageCarries(t *testing.T) {
	quoting := newQuoting(t, 10)
	quoting.sent(t, messages.NewMessage("legacy", draftedAt.Add(-time.Hour), messages.NoAccount, "fr", "hello").
		Named(messages.AuthorOf("guest_Bob", false, 0, 0, messages.Title{})))

	quote, err := quoting.quotes.Quote(t.Context(), "legacy")

	require.NoError(t, err)
	assert.Equal(t, "guest_Bob", quote.Author().Name())
	assert.Empty(t, quoting.namer.asked, "it has no account to read a name from")
}

func TestAQuoteNobodyCouldBeAskedAboutRefusesTheReply(t *testing.T) {
	quoting := newQuoting(t, 10)
	quoting.sent(t, messages.NewMessage("hello", draftedAt.Add(-time.Hour), ada, "fr", "hello planet"))
	quoting.namer.err = errors.New("the player module is stuck")

	_, err := quoting.quotes.Quote(t.Context(), "hello")

	require.ErrorIs(t, err, messages.ErrAuthorUnavailable)
}
