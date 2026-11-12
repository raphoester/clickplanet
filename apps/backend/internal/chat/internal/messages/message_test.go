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
		Room:    messages.FlagRoom("fr"),
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

func TestAMessageWithNoRoomIsNotRestored(t *testing.T) {
	_, err := messages.Snapshot{ID: "message-1", Text: "hello"}.Restore()

	require.ErrorIs(t, err, messages.ErrInvalidMessage)
}

func TestAnAnswerToAnAnswerQuotesThatOneAlone(t *testing.T) {
	question := messages.Restored(t, messages.Snapshot{ID: "message-1", Text: "who holds the north"})
	answer, err := messages.Restored(t, messages.Snapshot{ID: "message-2", Text: "we do"}).Answering(question)
	require.NoError(t, err)

	followUp, err := messages.Restored(t, messages.Snapshot{ID: "message-3", Text: "since when"}).Answering(answer)

	require.NoError(t, err)
	assert.Equal(t, messages.Quote{MessageID: "message-2", Text: "we do"}, followUp.TakeSnapshot().Quote)
}

func TestAnAnswerToAGifQuotesTheGif(t *testing.T) {
	question := messages.Restored(t, messages.Snapshot{ID: "message-1", Gif: dancing})

	answer, err := messages.Restored(t, messages.Snapshot{ID: "message-2", Text: "same"}).Answering(question)

	require.NoError(t, err)
	assert.Equal(t, messages.Quote{MessageID: "message-1", Gif: dancing}, answer.TakeSnapshot().Quote)
}
