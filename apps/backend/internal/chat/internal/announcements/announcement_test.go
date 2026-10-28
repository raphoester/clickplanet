package announcements_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
)

func payloadOf(t *testing.T, told func(time.Time) (announcements.Announcement, error)) string {
	t.Helper()

	announcement, err := told(at)
	require.NoError(t, err)
	return string(announcement.Payload())
}

func TestAMuteSaysWhoAndForHowManySeconds(t *testing.T) {
	assert.JSONEq(t, `{"name":"guest_a1b2c3","seconds":3600}`,
		payloadOf(t, announcements.MutedOf(mutes.MuteID{15: 1}, "guest_a1b2c3", time.Hour).Announcement))
}

func TestAMuteIsAKindTheChatKnows(t *testing.T) {
	assert.True(t, announcements.KindMute.Known())
}

func TestARoundNamesItsPodium(t *testing.T) {
	round := announcements.DayOf(0, 5).Placed([]announcements.Place{
		announcements.PlaceOf("fr", 1, 25),
		announcements.PlaceOf("de", 2, 18),
		announcements.PlaceOf("es", 3, 15),
		announcements.PlaceOf("it", 4, 12),
	})

	assert.JSONEq(t, `{"number":5,"podium":[
		{"country":"fr","rank":1,"points":25},
		{"country":"de","rank":2,"points":18},
		{"country":"es","rank":3,"points":15}
	]}`, payloadOf(t, round.Announcement))
}

func TestCountriesTiedOnThePodiumAreAllNamed(t *testing.T) {
	round := announcements.DayOf(0, 23).Placed([]announcements.Place{
		announcements.PlaceOf("fr", 1, 75),
		announcements.PlaceOf("de", 2, 54),
		announcements.PlaceOf("it", 2, 54),
		announcements.PlaceOf("es", 4, 36),
	})

	assert.JSONEq(t, `{"number":23,"podium":[
		{"country":"fr","rank":1,"points":75},
		{"country":"de","rank":2,"points":54},
		{"country":"it","rank":2,"points":54}
	]}`, payloadOf(t, round.Announcement))
}

func TestABattleSaysItIsARoundOfItsOwnAndItsTimes(t *testing.T) {
	round := announcements.BattleOf(0, 1, true, 3).Placed([]announcements.Place{announcements.PlaceOf("fr", 1, 75)})

	assert.JSONEq(t, `{"number":1,"finale":true,"own":true,"times":3,"podium":[
		{"country":"fr","rank":1,"points":75}
	]}`, payloadOf(t, round.Announcement))
}

func TestARoundNobodyHeldGroundInHasAnEmptyPodium(t *testing.T) {
	assert.JSONEq(t, `{"number":1,"podium":[]}`, payloadOf(t, announcements.DayOf(0, 1).Placed(nil).Announcement))
}

func TestAnEmptyIDIsRefused(t *testing.T) {
	_, err := announcements.RoundIDOf("")
	require.ErrorIs(t, err, announcements.ErrNoRound)
	_, err = announcements.BookIDOf("")
	require.ErrorIs(t, err, announcements.ErrNoBook)
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

func TestAFactIsAnnouncedUnderAnIDOfItsOwn(t *testing.T) {
	ids := map[announcements.AnnouncementID]string{}
	for name, told := range map[string]func(time.Time) (announcements.Announcement, error){
		"bomb 7":        announcements.BombOf(7, "fr", "de", 42, 3).Announcement,
		"bomb 8":        announcements.BombOf(8, "fr", "de", 42, 3).Announcement,
		"fortify 7":     announcements.FortifyOf(7, "fr", "is", 3, 60).Announcement,
		"mute 1":        announcements.MutedOf(mutes.MuteID{15: 1}, "guest_a1b2c3", time.Hour).Announcement,
		"day 5":         announcements.DayOf(0, 5).Announcement,
		"day 5 of 1":    announcements.DayOf(1, 5).Announcement,
		"battle 5":      announcements.BattleOf(0, 5, false, 2).Announcement,
		"champion of 0": announcements.ChampionOf(0, nil).Announcement,
	} {
		first, err := told(at)
		require.NoError(t, err)
		again, err := told(at.Add(time.Hour))
		require.NoError(t, err)

		assert.Equal(t, first.ID(), again.ID(), "%s told twice", name)
		ids[first.ID()] += name
	}

	assert.Len(t, ids, 8, "each fact has an id no other fact has")
}
