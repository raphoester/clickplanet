package conversations_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var (
	note  = notes.NoteID{15: 1}
	start = time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
)

func empty(t *testing.T) conversations.Conversation {
	t.Helper()

	conversation, err := conversations.Snapshot{Note: note}.Restore()
	require.NoError(t, err)
	return conversation
}

func TestAnAnswerIsKeptCleanedWithItsMailAndTime(t *testing.T) {
	answered, err := empty(t).Answered("  Thanks!\r\nWhich phone?  ", "a1@clickplanet.lol", start)
	require.NoError(t, err)

	assert.Equal(t, conversations.Snapshot{Note: note, Answers: []conversations.Answer{
		{Text: "Thanks!\nWhich phone?", Mail: "a1@clickplanet.lol", At: start},
	}}, answered.TakeSnapshot())
}

func TestAnAnswerWithNoWordsOrTooManyIsRefused(t *testing.T) {
	for _, text := range []string{" \n ", strings.Repeat("a", notes.MaxAnswerTextLength+1)} {
		_, err := empty(t).Answered(text, "a1@clickplanet.lol", start)

		require.ErrorIs(t, err, notes.ErrInvalidText)
	}
}

func TestAReplyIsKeptOnceWhateverItSays(t *testing.T) {
	reply := conversations.Reply{Mail: "r1@example.com", From: "ada@example.com", Text: "\x00An iPhone\t13 ", At: start}

	replied, err := empty(t).Replied(reply)
	require.NoError(t, err)
	_, err = replied.Replied(reply)

	require.ErrorIs(t, err, conversations.ErrReceivedAlready)
	assert.Equal(t, []conversations.Reply{{Mail: "r1@example.com", From: "ada@example.com", Text: "An iPhone 13", At: start}},
		replied.TakeSnapshot().Replies)
}

func TestTheThreadIsEveryMailOldestFirst(t *testing.T) {
	conversation, err := conversations.Snapshot{
		Note: note,
		Answers: []conversations.Answer{
			{Text: "Which phone?", Mail: "a1@clickplanet.lol", At: start},
			{Text: "Fixed", Mail: "a2@clickplanet.lol", At: start.Add(2 * time.Hour)},
		},
		Replies: []conversations.Reply{{Mail: "r1@example.com", Text: "An iPhone", At: start.Add(time.Hour)}},
	}.Restore()
	require.NoError(t, err)

	assert.Equal(t, []cpmail.MessageID{"a1@clickplanet.lol", "r1@example.com", "a2@clickplanet.lol"}, conversation.Thread())
	assert.Empty(t, empty(t).Thread())
}

func TestAClosedConversationKeepsItsMailsAndAnAnonymizedOneItsWordsWithNoSender(t *testing.T) {
	replied, err := empty(t).Replied(conversations.Reply{Mail: "r1@example.com", From: "ada@example.com", Text: "Hi", At: start})
	require.NoError(t, err)

	closed := replied.Closed(start.Add(time.Hour)).Anonymized().TakeSnapshot()

	assert.Equal(t, conversations.Snapshot{
		Note: note, Replies: []conversations.Reply{{Mail: "r1@example.com", Text: "Hi", At: start}}, ClosedAt: start.Add(time.Hour),
	}, closed)
	assert.Equal(t, cpmail.Address("ada@example.com"), replied.TakeSnapshot().Replies[0].From, "a copy, the conversation stays")
}

func TestASnapshotThatBreaksARuleRestoresNothing(t *testing.T) {
	for name, broken := range map[string]conversations.Snapshot{
		"no note":       {},
		"a bad mail id": {Note: note, Replies: []conversations.Reply{{Mail: "a b", Text: "Hi", At: start}}},
		"no time":       {Note: note, Answers: []conversations.Answer{{Text: "Hi", Mail: "a1@clickplanet.lol"}}},
	} {
		_, err := broken.Restore()

		require.ErrorIs(t, err, conversations.ErrInvalidConversation, name)
	}
}

func TestAReplyIsCutPastItsLengthNotRefused(t *testing.T) {
	assert.Len(t, []rune(conversations.ReplyTextOf(strings.Repeat("é", conversations.MaxReplyTextLength+5))), conversations.MaxReplyTextLength)
	assert.Equal(t, "line\nnext", conversations.ReplyTextOf("line\r\nnext\x07"))
	assert.Equal(t, "a�b", conversations.ReplyTextOf("a\xffb"))
}
