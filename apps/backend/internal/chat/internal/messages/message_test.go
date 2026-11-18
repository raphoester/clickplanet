package messages_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

func TestAMessageWithNoIDIsNotRestored(t *testing.T) {
	_, err := messages.Snapshot{Text: "hello"}.Restore()

	require.ErrorIs(t, err, messages.ErrInvalidMessage)
}

func TestAMessageWithNoRoomIsNotRestored(t *testing.T) {
	_, err := messages.Snapshot{ID: "message-1", Text: "hello"}.Restore()

	require.ErrorIs(t, err, messages.ErrInvalidMessage)
}

func TestAnAnswerToAGifQuotesTheGif(t *testing.T) {
	question := messages.Restored(t, messages.Snapshot{ID: "message-1", Gif: dancing})

	answer, err := messages.Restored(t, messages.Snapshot{ID: "message-2", Text: "same"}).Answering(question)

	require.NoError(t, err)
	assert.Equal(t, messages.Quote{MessageID: "message-1", Gif: dancing}, answer.TakeSnapshot().Quote)
}
