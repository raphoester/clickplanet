package get_history_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/inmemory_message_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/get_history_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/inmemory_reaction_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type stubAuthors struct {
	author messages.Author
	err    error
}

func (s stubAuthors) Author(context.Context, messages.AccountID, string) (messages.Author, error) {
	return s.author, s.err
}

var (
	ada    = cpsession.AccountID{15: 1}
	now    = time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	clown  = reactions.Reaction(2)
	player = messages.Author{Username: "ada", Tag: "a1b2c3"}
)

type fixture struct {
	messages  *inmemory_message_storage.Storage
	reactions *inmemory_reaction_storage.Storage
}

func newFixture(t *testing.T, texts ...string) fixture {
	t.Helper()

	f := fixture{messages: inmemory_message_storage.New(), reactions: inmemory_reaction_storage.New()}
	for i, text := range texts {
		require.NoError(t, f.messages.Append(t.Context(), messages.Record{Message: messages.Message{
			ID: messages.MessageID(text), Text: text, SentAt: now.Add(time.Duration(i-len(texts)) * time.Hour),
		}}))
	}
	return f
}

func (f fixture) history(t *testing.T, authors stubAuthors, account messages.AccountID) []get_history_usecase.Entry {
	t.Helper()

	history, err := get_history_usecase.New(f.messages, f.reactions, authors, cptime.NewFixedClock(now),
		messages.Window{Size: 2, Retention: 24 * time.Hour}).Execute(t.Context(), account)
	require.NoError(t, err)
	return history
}

func texts(history []get_history_usecase.Entry) []string {
	texts := make([]string, 0, len(history))
	for _, entry := range history {
		texts = append(texts, entry.Message.Text)
	}
	return texts
}

func TestTheHistoryIsTheWindowOfNewestMessages(t *testing.T) {
	f := newFixture(t, "old", "middle", "new")

	assert.Equal(t, []string{"middle", "new"}, texts(f.history(t, stubAuthors{author: player}, ada)))
}

func TestEachMessageCarriesItsReactionsMarkedForTheCaller(t *testing.T) {
	f := newFixture(t, "hello")
	require.NoError(t, f.reactions.Save(t.Context(), reactions.Change{
		MessageID: "hello", Reaction: clown, Reactor: reactions.ReactorOf(ada, player), On: true, At: now,
	}))

	assert.Equal(t, []reactions.Count{{Reaction: clown, Count: 1, Mine: true}},
		f.history(t, stubAuthors{author: player}, ada)[0].Reactions)
	assert.Equal(t, uint64(1), f.history(t, stubAuthors{author: player}, ada)[0].ReactionsVersion)
	assert.Equal(t, []reactions.Count{{Reaction: clown, Count: 1, Mine: false}},
		f.history(t, stubAuthors{author: player}, cpsession.NoAccount)[0].Reactions, "a guest is its tag, not the account")
}

func TestAHistoryIsServedWhenThePlayerModuleDoesNotAnswer(t *testing.T) {
	f := newFixture(t, "hello")
	require.NoError(t, f.reactions.Save(t.Context(), reactions.Change{
		MessageID: "hello", Reaction: clown, Reactor: reactions.ReactorOf(ada, player), On: true, At: now,
	}))

	history := f.history(t, stubAuthors{err: assert.AnError}, ada)

	assert.Equal(t, []reactions.Count{{Reaction: clown, Count: 1, Mine: false}}, history[0].Reactions)
}
