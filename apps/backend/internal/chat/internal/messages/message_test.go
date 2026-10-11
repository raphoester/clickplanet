package messages_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

func TestAMessageIsRestoredAsItsSnapshotSays(t *testing.T) {
	kept := messages.Snapshot{
		ID:      "message-2",
		SentAt:  time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC),
		Account: messages.AccountID{15: 1},
		Author:  messages.AuthorOf("Ada_L", true, 7, 12, messages.Title{}),
		Country: "fr",
		Text:    "we do",
		Quote:   messages.Quote{MessageID: "message-1"},
	}

	message, err := kept.Restore()

	require.NoError(t, err)
	assert.Equal(t, kept, message.TakeSnapshot())
}

func TestAMessageWithNoIDIsNotRestored(t *testing.T) {
	_, err := messages.Snapshot{Text: "hello"}.Restore()

	require.ErrorIs(t, err, messages.ErrInvalidMessage)
}

func TestAnAnswerQuotesTheMessageItAnswersAndIsOtherwiseUnchanged(t *testing.T) {
	bob := messages.AuthorOf("Bob", false, 3, 0, messages.Title{})
	question := messages.Restored(t, messages.Snapshot{
		ID: "message-1", Account: messages.AccountID{15: 2}, Author: bob, Country: "de", Text: "who holds the north",
	})
	draft := messages.Snapshot{ID: "message-2", Account: messages.AccountID{15: 1}, Country: "fr", Text: "we do"}

	answer := messages.Restored(t, draft).Answering(question)

	draft.Quote = messages.Quote{MessageID: "message-1", Author: bob, Text: "who holds the north"}
	assert.Equal(t, draft, answer.TakeSnapshot())
}

func TestAnAnswerToAnAnswerQuotesThatOneAlone(t *testing.T) {
	question := messages.Restored(t, messages.Snapshot{ID: "message-1", Text: "who holds the north"})
	answer := messages.Restored(t, messages.Snapshot{ID: "message-2", Text: "we do"}).Answering(question)

	followUp := messages.Restored(t, messages.Snapshot{ID: "message-3", Text: "since when"}).Answering(answer)

	assert.Equal(t, messages.Quote{MessageID: "message-2", Text: "we do"}, followUp.TakeSnapshot().Quote)
}
