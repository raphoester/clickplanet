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

func TestARoundNamesItsPodium(t *testing.T) {
	payload, err := announcements.RoundOf(5, false, []announcements.Place{
		announcements.PlaceOf("fr", 1, 25),
		announcements.PlaceOf("de", 2, 18),
		announcements.PlaceOf("es", 3, 15),
		announcements.PlaceOf("it", 4, 12),
	}).Payload()

	require.NoError(t, err)
	assert.JSONEq(t, `{"number":5,"podium":[
		{"country":"fr","rank":1,"points":25},
		{"country":"de","rank":2,"points":18},
		{"country":"es","rank":3,"points":15}
	]}`, string(payload))
}

func TestCountriesTiedOnThePodiumAreAllNamed(t *testing.T) {
	payload, err := announcements.RoundOf(23, true, []announcements.Place{
		announcements.PlaceOf("fr", 1, 75),
		announcements.PlaceOf("de", 2, 54),
		announcements.PlaceOf("it", 2, 54),
		announcements.PlaceOf("es", 4, 36),
	}).Payload()

	require.NoError(t, err)
	assert.JSONEq(t, `{"number":23,"finale":true,"podium":[
		{"country":"fr","rank":1,"points":75},
		{"country":"de","rank":2,"points":54},
		{"country":"it","rank":2,"points":54}
	]}`, string(payload))
}

func TestARoundNobodyHeldGroundInHasAnEmptyPodium(t *testing.T) {
	payload, err := announcements.RoundOf(1, false, nil).Payload()

	require.NoError(t, err)
	assert.JSONEq(t, `{"number":1,"podium":[]}`, string(payload))
}

func TestARoundIsAKindTheChatKnows(t *testing.T) {
	assert.True(t, announcements.KindRound.Known())
}
