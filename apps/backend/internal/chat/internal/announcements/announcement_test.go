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

var at = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)

func TestTheChampionIsTheCountryFirstInTheSeasonsTable(t *testing.T) {
	champion := announcements.ChampionOf(0, []announcements.Place{
		announcements.PlaceOf("de", 1, 118),
		announcements.PlaceOf("fr", 2, 97),
	})

	announcement, err := champion.Announcement(at)

	require.NoError(t, err)
	assert.True(t, champion.Crowned())
	assert.Equal(t, announcements.KindChampion, announcement.Kind())
	assert.Equal(t, at, announcement.At())
	assert.JSONEq(t, `{"season":0,"countries":["de"]}`, string(announcement.Payload()))
}

func TestCountriesLevelAtTheTopAreChampionsTogether(t *testing.T) {
	announcement, err := announcements.ChampionOf(2, []announcements.Place{
		announcements.PlaceOf("de", 1, 100),
		announcements.PlaceOf("fr", 1, 100),
		announcements.PlaceOf("it", 3, 40),
	}).Announcement(at)

	require.NoError(t, err)
	assert.JSONEq(t, `{"season":2,"countries":["de","fr"]}`, string(announcement.Payload()))
}

func TestASeasonNobodyScoredInHasNoChampion(t *testing.T) {
	assert.False(t, announcements.ChampionOf(0, nil).Crowned())
}

func TestAChampionIsAnnouncedUnderItsSeasonsID(t *testing.T) {
	first, err := announcements.ChampionOf(0, []announcements.Place{announcements.PlaceOf("de", 1, 118)}).Announcement(at)
	require.NoError(t, err)
	again, err := announcements.ChampionOf(0, []announcements.Place{announcements.PlaceOf("fr", 1, 97)}).Announcement(at.Add(time.Hour))
	require.NoError(t, err)
	next, err := announcements.ChampionOf(1, []announcements.Place{announcements.PlaceOf("de", 1, 118)}).Announcement(at)
	require.NoError(t, err)

	assert.Equal(t, first.ID(), again.ID())
	assert.NotEqual(t, first.ID(), next.ID())
}
