package messages_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

func TestTheSameSenderAlwaysGetsTheSameTag(t *testing.T) {
	first := messages.Tag("pepper", "1.2.3.4")
	second := messages.Tag("pepper", "1.2.3.4")

	assert.Equal(t, first, second)
}

func TestDifferentSendersGetDifferentTags(t *testing.T) {
	assert.NotEqual(t, messages.Tag("pepper", "1.2.3.4"), messages.Tag("pepper", "5.6.7.8"))
}

func TestTheSaltChangesTheTag(t *testing.T) {
	assert.NotEqual(t, messages.Tag("pepper", "1.2.3.4"), messages.Tag("other pepper", "1.2.3.4"))
}

func TestTheTagDoesNotContainTheAddress(t *testing.T) {
	assert.NotContains(t, messages.Tag("pepper", "1.2.3.4"), "1.2.3.4")
}
