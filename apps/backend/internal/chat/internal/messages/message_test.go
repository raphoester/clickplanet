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

func TestAnAnswerQuotesTheMessageItAnswersAndIsOtherwiseUnchanged(t *testing.T) {
	bob := messages.AuthorOf("Bob", false, 3, 0, messages.Title{})
	question := messages.Restored(t, messages.Snapshot{
		ID: "message-1", Account: messages.AccountID{15: 2}, Author: bob, Country: "de", Text: "who holds the north",
	})
	draft := messages.Snapshot{
		ID: "message-2", Account: messages.AccountID{15: 1}, Room: messages.World, Country: "fr", Text: "we do",
	}

	answer, err := messages.Restored(t, draft).Answering(question)

	require.NoError(t, err)
	draft.Quote = messages.Quote{MessageID: "message-1", Author: bob, Text: "who holds the north"}
	assert.Equal(t, draft, answer.TakeSnapshot())
}

func TestAnAnswerToAnAnswerQuotesThatOneAlone(t *testing.T) {
	question := messages.Restored(t, messages.Snapshot{ID: "message-1", Text: "who holds the north"})
	answer, err := messages.Restored(t, messages.Snapshot{ID: "message-2", Text: "we do"}).Answering(question)
	require.NoError(t, err)

	followUp, err := messages.Restored(t, messages.Snapshot{ID: "message-3", Text: "since when"}).Answering(answer)

	require.NoError(t, err)
	assert.Equal(t, messages.Quote{MessageID: "message-2", Text: "we do"}, followUp.TakeSnapshot().Quote)
}

func TestAMessageOfAnotherRoomCannotBeAnswered(t *testing.T) {
	question := messages.Restored(t, messages.Snapshot{ID: "message-1", Room: messages.FlagRoom("de"), Text: "where now"})

	_, err := messages.Restored(t, messages.Snapshot{ID: "message-2", Room: messages.FlagRoom("fr"), Text: "north"}).
		Answering(question)

	require.ErrorIs(t, err, messages.ErrNotShown)
}

func TestAnAnswerToAGifQuotesTheGif(t *testing.T) {
	question := messages.Restored(t, messages.Snapshot{ID: "message-1", Gif: dancing})

	answer, err := messages.Restored(t, messages.Snapshot{ID: "message-2", Text: "same"}).Answering(question)

	require.NoError(t, err)
	assert.Equal(t, messages.Quote{MessageID: "message-1", Gif: dancing}, answer.TakeSnapshot().Quote)
}

func TestAGuestSendsATextButNoGif(t *testing.T) {
	guest := messages.GuestOf("guest_0b1c2d", 3)

	text, err := messages.Restored(t, messages.Snapshot{ID: "message-1", Text: "hello"}).SentBy(guest)
	require.NoError(t, err)
	assert.Equal(t, guest, text.TakeSnapshot().Author)

	_, err = messages.Restored(t, messages.Snapshot{ID: "message-2", Gif: dancing}).SentBy(guest)
	require.ErrorIs(t, err, messages.ErrGuestGif)
}

func TestAPlayerSendsAGif(t *testing.T) {
	ada := messages.AuthorOf("Ada_L", false, 7, 12, messages.Title{})

	sent, err := messages.Restored(t, messages.Snapshot{ID: "message-1", Gif: dancing}).SentBy(ada)

	require.NoError(t, err)
	assert.Equal(t, ada, sent.TakeSnapshot().Author)
}
