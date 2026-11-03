package messages_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

func TestNoFlagOnTheWireIsTheChatEveryoneReads(t *testing.T) {
	assert.Equal(t, messages.World, messages.RoomOf(""))
	assert.Equal(t, messages.FlagRoom("fr"), messages.RoomOf("fr"))
}

func TestEveryoneEntersTheChatEveryoneReadsWithNoQuestionAsked(t *testing.T) {
	rooms := messages.NewRooms(messages.FailingFlags{})

	require.NoError(t, rooms.EntryError(t.Context(), messages.NoAccount, messages.World))
	require.NoError(t, rooms.EntryError(t.Context(), ada, messages.World))
}

func TestOnlyTheAccountsThatPlayForAFlagEnterItsRoom(t *testing.T) {
	rooms := messages.NewRooms(messages.FakeFlags{ada: "fr", bob: "de"})

	require.NoError(t, rooms.EntryError(t.Context(), ada, messages.FlagRoom("fr")))
	require.ErrorIs(t, rooms.EntryError(t.Context(), bob, messages.FlagRoom("fr")), messages.ErrNotInRoom)
	require.ErrorIs(t, rooms.EntryError(t.Context(), messages.AccountID{15: 9}, messages.FlagRoom("fr")),
		messages.ErrNotInRoom, "an account that never pledged a flag")
	require.ErrorIs(t, rooms.EntryError(t.Context(), messages.NoAccount, messages.FlagRoom("fr")), messages.ErrNoAccount)
}

func TestAFlagNobodyCouldReadKeepsTheRoomShut(t *testing.T) {
	err := messages.NewRooms(messages.FailingFlags{}).EntryError(t.Context(), ada, messages.FlagRoom("fr"))

	require.ErrorIs(t, err, messages.ErrFlagsDown)
	assert.NotErrorIs(t, err, messages.ErrNotInRoom)
}
