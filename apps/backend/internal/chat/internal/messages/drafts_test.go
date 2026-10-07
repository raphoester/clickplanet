package messages_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	draftedAt = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	ada       = messages.AccountID{15: 1}
)

type failingIDs struct{}

func (failingIDs) NewID() (messages.MessageID, error) { return "", errors.New("no entropy") }

func drafts(ids messages.IDProvider) messages.Drafts {
	return messages.NewDrafts(ids, cptime.NewFixedClock(draftedAt), limits)
}

func TestADraftIsCleanedAndStampedWithAnIDAndTheTime(t *testing.T) {
	message, err := drafts(&messages.SequentialIDs{}).Draft(ada, "fr", "  hello\tplanet  ")

	require.NoError(t, err)
	assert.Equal(t,
		messages.NewMessage("00000000-0000-0000-0000-000000000001", draftedAt, ada, "fr", "hello planet"),
		message)
}

func TestEachDraftGetsAnIDOfItsOwn(t *testing.T) {
	drafting := drafts(&messages.SequentialIDs{})

	first, err := drafting.Draft(ada, "fr", "one")
	require.NoError(t, err)
	second, err := drafting.Draft(ada, "fr", "two")
	require.NoError(t, err)

	assert.Equal(t, messages.MessageID("00000000-0000-0000-0000-000000000001"), first.ID())
	assert.Equal(t, messages.MessageID("00000000-0000-0000-0000-000000000002"), second.ID())
}

func TestAnInvalidTextIsRefusedBeforeAnIDIsDrawn(t *testing.T) {
	drafting := drafts(&messages.SequentialIDs{})

	_, err := drafting.Draft(ada, "fr", "   ")
	require.ErrorIs(t, err, messages.ErrInvalidMessage)

	message, err := drafting.Draft(ada, "fr", "hello")
	require.NoError(t, err)
	assert.Equal(t, messages.MessageID("00000000-0000-0000-0000-000000000001"), message.ID())
}

func TestADraftWithNoIDFails(t *testing.T) {
	_, err := drafts(failingIDs{}).Draft(ada, "fr", "hello")

	require.Error(t, err)
	assert.NotErrorIs(t, err, messages.ErrInvalidMessage)
}
