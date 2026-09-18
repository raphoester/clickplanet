package get_history_usecase_test

import (
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

var (
	ada   = cpsession.AccountID{15: 1}
	now   = time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	clown = reactions.Reaction(2)
	other = cpsession.AccountID{15: 2}
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

func (f fixture) history(t *testing.T, account messages.AccountID) []get_history_usecase.Entry {
	t.Helper()

	history, err := get_history_usecase.New(f.messages, f.reactions, cptime.NewFixedClock(now),
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

	assert.Equal(t, []string{"middle", "new"}, texts(f.history(t, ada)))
}

func TestEachMessageCarriesItsReactionsMarkedForTheCallersAccount(t *testing.T) {
	f := newFixture(t, "hello")
	require.NoError(t, f.reactions.Save(t.Context(), reactions.Change{
		MessageID: "hello", Reaction: clown, Reactor: reactions.ReactorOf(ada), On: true, At: now,
	}))

	assert.Equal(t, []reactions.Count{{Reaction: clown, Count: 1, Mine: true}}, f.history(t, ada)[0].Reactions)
	assert.Equal(t, uint64(1), f.history(t, ada)[0].ReactionsVersion)
	assert.Equal(t, []reactions.Count{{Reaction: clown, Count: 1, Mine: false}},
		f.history(t, other)[0].Reactions, "another account")
	assert.Equal(t, []reactions.Count{{Reaction: clown, Count: 1, Mine: false}},
		f.history(t, cpsession.NoAccount)[0].Reactions, "no token")
}
