package messages_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	draftedAt = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	ada       = messages.AccountID{15: 1}
)

type failingIDs struct{}

func (failingIDs) NewID() (messages.MessageID, error) { return "", errors.New("no entropy") }

func drafts(ids messages.IDProvider) messages.Drafts {
	return messages.NewDrafts(ids, cptime.NewFixedClock(draftedAt), limits, cpcountries.New(),
		messages.NewRooms(messages.FakeFlags{ada: "fr"}))
}

func TestADraftIsCleanedAndStampedWithAnIDAndTheTime(t *testing.T) {
	message, err := drafts(&messages.SequentialIDs{}).Draft(t.Context(), ada, messages.World, "fr", "  hello\tplanet  ")

	require.NoError(t, err)
	assert.Equal(t,
		messages.Snapshot{
			ID: "00000000-0000-0000-0000-000000000001", SentAt: draftedAt, Account: ada, Room: messages.World, Country: "fr",
			Text: "hello planet",
		},
		message.TakeSnapshot())
}

func TestEachDraftGetsAnIDOfItsOwn(t *testing.T) {
	drafting := drafts(&messages.SequentialIDs{})

	first, err := drafting.Draft(t.Context(), ada, messages.World, "fr", "one")
	require.NoError(t, err)
	second, err := drafting.Draft(t.Context(), ada, messages.World, "fr", "two")
	require.NoError(t, err)

	assert.Equal(t, messages.MessageID("00000000-0000-0000-0000-000000000001"), first.TakeSnapshot().ID)
	assert.Equal(t, messages.MessageID("00000000-0000-0000-0000-000000000002"), second.TakeSnapshot().ID)
}

func TestAnInvalidTextIsRefusedBeforeAnIDIsDrawn(t *testing.T) {
	drafting := drafts(&messages.SequentialIDs{})

	_, err := drafting.Draft(t.Context(), ada, messages.World, "fr", "   ")
	require.ErrorIs(t, err, messages.ErrInvalidMessage)

	message, err := drafting.Draft(t.Context(), ada, messages.World, "fr", "hello")
	require.NoError(t, err)
	assert.Equal(t, messages.MessageID("00000000-0000-0000-0000-000000000001"), message.TakeSnapshot().ID)
}

func TestAnUnknownCountryIsRefusedBeforeAnIDIsDrawn(t *testing.T) {
	drafting := drafts(&messages.SequentialIDs{})

	_, err := drafting.Draft(t.Context(), ada, messages.World, "atlantis", "hello")
	require.ErrorIs(t, err, messages.ErrInvalidMessage)

	message, err := drafting.Draft(t.Context(), ada, messages.World, "fr", "hello")
	require.NoError(t, err)
	assert.Equal(t, messages.MessageID("00000000-0000-0000-0000-000000000001"), message.TakeSnapshot().ID)
}

func TestADraftWithNoIDFails(t *testing.T) {
	_, err := drafts(failingIDs{}).Draft(t.Context(), ada, messages.World, "fr", "hello")

	require.Error(t, err)
	assert.NotErrorIs(t, err, messages.ErrInvalidMessage)
}

func TestADraftGoesToTheRoomOfTheFlagItsAccountPlaysFor(t *testing.T) {
	message, err := drafts(&messages.SequentialIDs{}).Draft(t.Context(), ada, messages.FlagRoom("fr"), "fr", "hello team")

	require.NoError(t, err)
	assert.True(t, message.In(messages.FlagRoom("fr")))
}

func TestADraftForTheRoomOfAnotherFlagIsRefusedBeforeAnIDIsDrawn(t *testing.T) {
	drafting := drafts(&messages.SequentialIDs{})

	_, err := drafting.Draft(t.Context(), ada, messages.FlagRoom("de"), "fr", "hello")
	require.ErrorIs(t, err, messages.ErrNotInRoom)

	message, err := drafting.Draft(t.Context(), ada, messages.World, "fr", "hello")
	require.NoError(t, err)
	assert.Equal(t, messages.MessageID("00000000-0000-0000-0000-000000000001"), message.TakeSnapshot().ID)
}
