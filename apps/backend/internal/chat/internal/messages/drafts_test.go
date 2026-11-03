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
	message, err := drafts(&messages.SequentialIDs{}).Draft(t.Context(), ada, messages.World, "fr", "  hello\tplanet  ", messages.Gif{})

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

	first, err := drafting.Draft(t.Context(), ada, messages.World, "fr", "one", messages.Gif{})
	require.NoError(t, err)
	second, err := drafting.Draft(t.Context(), ada, messages.World, "fr", "two", messages.Gif{})
	require.NoError(t, err)

	assert.Equal(t, messages.MessageID("00000000-0000-0000-0000-000000000001"), first.TakeSnapshot().ID)
	assert.Equal(t, messages.MessageID("00000000-0000-0000-0000-000000000002"), second.TakeSnapshot().ID)
}

func TestAnInvalidTextIsRefusedBeforeAnIDIsDrawn(t *testing.T) {
	drafting := drafts(&messages.SequentialIDs{})

	_, err := drafting.Draft(t.Context(), ada, messages.World, "fr", "   ", messages.Gif{})
	require.ErrorIs(t, err, messages.ErrInvalidMessage)

	message, err := drafting.Draft(t.Context(), ada, messages.World, "fr", "hello", messages.Gif{})
	require.NoError(t, err)
	assert.Equal(t, messages.MessageID("00000000-0000-0000-0000-000000000001"), message.TakeSnapshot().ID)
}

func TestAnUnknownCountryIsRefusedBeforeAnIDIsDrawn(t *testing.T) {
	drafting := drafts(&messages.SequentialIDs{})

	_, err := drafting.Draft(t.Context(), ada, messages.World, "atlantis", "hello", messages.Gif{})
	require.ErrorIs(t, err, messages.ErrInvalidMessage)

	message, err := drafting.Draft(t.Context(), ada, messages.World, "fr", "hello", messages.Gif{})
	require.NoError(t, err)
	assert.Equal(t, messages.MessageID("00000000-0000-0000-0000-000000000001"), message.TakeSnapshot().ID)
}

func TestADraftWithNoIDFails(t *testing.T) {
	_, err := drafts(failingIDs{}).Draft(t.Context(), ada, messages.World, "fr", "hello", messages.Gif{})

	require.Error(t, err)
	assert.NotErrorIs(t, err, messages.ErrInvalidMessage)
}

func TestADraftGoesToTheRoomOfTheFlagItsAccountPlaysFor(t *testing.T) {
	message, err := drafts(&messages.SequentialIDs{}).Draft(t.Context(), ada, messages.FlagRoom("fr"), "fr", "hello team", messages.Gif{})

	require.NoError(t, err)
	assert.True(t, message.In(messages.FlagRoom("fr")))
}

func TestADraftForTheRoomOfAnotherFlagIsRefusedBeforeAnIDIsDrawn(t *testing.T) {
	drafting := drafts(&messages.SequentialIDs{})

	_, err := drafting.Draft(t.Context(), ada, messages.FlagRoom("de"), "fr", "hello", messages.Gif{})
	require.ErrorIs(t, err, messages.ErrNotInRoom)

	message, err := drafting.Draft(t.Context(), ada, messages.World, "fr", "hello", messages.Gif{})
	require.NoError(t, err)
	assert.Equal(t, messages.MessageID("00000000-0000-0000-0000-000000000001"), message.TakeSnapshot().ID)
}

var dancing = messages.Gif{URL: "https://static.klipy.com/ii/935d7ab9/14/af/SE72470w.webp", Width: 220, Height: 220}

func TestADraftShowsAGifInsteadOfAText(t *testing.T) {
	message, err := drafts(&messages.SequentialIDs{}).Draft(t.Context(), ada, messages.World, "fr", "", dancing)

	require.NoError(t, err)
	assert.Equal(t, dancing, message.TakeSnapshot().Gif)
	assert.Empty(t, message.TakeSnapshot().Text)
}

func TestADraftWithAGifAndATextIsRefused(t *testing.T) {
	_, err := drafts(&messages.SequentialIDs{}).Draft(t.Context(), ada, messages.World, "fr", "look", dancing)

	require.ErrorIs(t, err, messages.ErrInvalidMessage)
}

func TestAGifFromAnywhereButKlipyIsRefused(t *testing.T) {
	for name, address := range map[string]string{
		"another host":       "https://evil.example/a.webp",
		"a host ending like": "https://static.klipy.com.evil.example/a.webp",
		"plain http":         "http://static.klipy.com/a.webp",
		"a user name":        "https://static.klipy.com@evil.example/a.webp",
		"a port":             "https://static.klipy.com:8443/a.webp",
		"no address":         "javascript:alert(1)",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := drafts(&messages.SequentialIDs{}).Draft(t.Context(), ada, messages.World, "fr", "",
				messages.Gif{URL: address, Width: 220, Height: 220})

			require.ErrorIs(t, err, messages.ErrInvalidMessage)
		})
	}
}

func TestAGifOfNoSizeOrAHugeOneIsRefused(t *testing.T) {
	for name, size := range map[string][2]uint32{"no width": {0, 220}, "no height": {220, 0}, "huge": {220, 5000}} {
		t.Run(name, func(t *testing.T) {
			gif := dancing
			gif.Width, gif.Height = size[0], size[1]

			_, err := drafts(&messages.SequentialIDs{}).Draft(t.Context(), ada, messages.World, "fr", "", gif)

			require.ErrorIs(t, err, messages.ErrInvalidMessage)
		})
	}
}
