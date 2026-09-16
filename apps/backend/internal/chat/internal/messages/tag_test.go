package messages_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

var tagger = messages.NewTagger("pepper")

func TestTheSameSenderAlwaysGetsTheSameTag(t *testing.T) {
	first := tagger.Of("1.2.3.4")

	assert.Equal(t, first, tagger.Of("1.2.3.4"))
}

func TestDifferentSendersGetDifferentTags(t *testing.T) {
	assert.NotEqual(t, tagger.Of("1.2.3.4"), tagger.Of("5.6.7.8"))
}

func TestTheSaltChangesTheTag(t *testing.T) {
	assert.NotEqual(t, tagger.Of("1.2.3.4"), messages.NewTagger("other pepper").Of("1.2.3.4"))
}

func TestTheTagDoesNotContainTheAddress(t *testing.T) {
	assert.NotContains(t, tagger.Of("1.2.3.4"), "1.2.3.4")
}

func TestAnIPv6SenderKeepsItsTagAcrossTheHostPart(t *testing.T) {
	assert.Equal(t, tagger.Of("2001:db8:1:2::1"), tagger.Of("2001:db8:1:2:dead:beef:0:9"))
}

func TestANeighbouringIPv6PrefixIsAnotherSender(t *testing.T) {
	assert.NotEqual(t, tagger.Of("2001:db8:1:2::1"), tagger.Of("2001:db8:1:3::1"))
}

func TestAParsedTagIsWhatTheChatShows(t *testing.T) {
	stamped := tagger.Of("1.2.3.4")

	parsed, err := messages.ParseTag("#" + stamped)

	require.NoError(t, err)
	assert.Equal(t, stamped, parsed)
}

func TestATagIsReadWithoutItsHashAndInAnyCase(t *testing.T) {
	parsed, err := messages.ParseTag("  A1B2C3 ")

	require.NoError(t, err)
	assert.Equal(t, "a1b2c3", parsed)
}

func TestSomethingThatIsNotATagIsRefused(t *testing.T) {
	for _, text := range []string{"", "a1b2c", "a1b2c3d", "zzzzzz", "#", "a1b2 c3"} {
		_, err := messages.ParseTag(text)
		assert.ErrorIs(t, err, messages.ErrInvalidTag, "%q", text)
	}
}
