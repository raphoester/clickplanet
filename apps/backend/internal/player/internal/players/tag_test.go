package players_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

func TestTheTagIsTheOneTheChatShowedBeforeItMoved(t *testing.T) {
	assert.Equal(t, players.Tag("2fe1b6"), players.TagOf("pepper", "1.2.3.4"))
}

func TestDifferentAddressesGetDifferentTags(t *testing.T) {
	assert.NotEqual(t, players.TagOf("pepper", "1.2.3.4"), players.TagOf("pepper", "5.6.7.8"))
}

func TestTheSaltChangesTheTag(t *testing.T) {
	assert.NotEqual(t, players.TagOf("pepper", "1.2.3.4"), players.TagOf("other pepper", "1.2.3.4"))
}

func TestTheTagDoesNotContainTheAddress(t *testing.T) {
	assert.NotContains(t, string(players.TagOf("pepper", "1.2.3.4")), "1.2.3.4")
}
