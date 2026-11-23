package messages_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	draftedAt = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	ada       = messages.AccountID{15: 1}
)

type failingIDs struct{}

func (failingIDs) NewID() (messages.MessageID, error) { return "", errors.New("no entropy") }

func drafts(ids messages.IDProvider) messages.Drafts {
	return messages.NewDrafts(ids, cptime.NewFixedClock(draftedAt), limits, cpflags.NewOpenCatalog(),
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

func TestADraftWithNoIDFails(t *testing.T) {
	_, err := drafts(failingIDs{}).Draft(t.Context(), ada, messages.World, "fr", "hello", messages.Gif{})

	require.Error(t, err)
	assert.NotErrorIs(t, err, messages.ErrInvalidMessage)
}

var dancing = messages.Gif{URL: "https://static.klipy.com/ii/935d7ab9/14/af/SE72470w.webp", Width: 220, Height: 220}

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

func TestAMessageInTheRoomOfAFlagOpenedWhileTheServerRunsIsDrafted(t *testing.T) {
	const rennes = "r-k3p9x2ma0b"
	catalog := cpflags.NewCatalog(cpflags.NewMemoryStore(), cpflags.Quiet{})
	drafts := messages.NewDrafts(&messages.SequentialIDs{}, cptime.NewFixedClock(draftedAt), limits, catalog,
		messages.NewRooms(messages.FakeFlags{ada: rennes}))
	room := messages.FlagRoom(rennes)

	_, err := drafts.Draft(t.Context(), ada, room, rennes, "kenavo", messages.Gif{})
	require.ErrorIs(t, err, messages.ErrInvalidMessage)

	require.NoError(t, catalog.Open(t.Context(), rennes, draftedAt.Add(-time.Hour)))
	message, err := drafts.Draft(t.Context(), ada, room, rennes, "kenavo", messages.Gif{})
	require.NoError(t, err)
	assert.Equal(t, room, message.TakeSnapshot().Room)

	require.NoError(t, catalog.Close(t.Context(), rennes, draftedAt))
	_, err = drafts.Draft(t.Context(), ada, room, rennes, "kenavo", messages.Gif{})
	require.NoError(t, err, "a closed flag is known: what its room does at a close is the next step's")
}
