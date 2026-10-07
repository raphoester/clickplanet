package announcements_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
)

func TestAMuteSaysWhoAndForHowManySeconds(t *testing.T) {
	payload, err := announcements.MutedOf("guest_a1b2c3", time.Hour).Payload()

	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"guest_a1b2c3","seconds":3600}`, string(payload))
}

func TestAMuteIsAKindTheChatKnows(t *testing.T) {
	assert.True(t, announcements.KindMute.Known())
}
