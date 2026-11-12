package announcements_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
)

func payloadOf(t *testing.T, told func(time.Time) (announcements.Announcement, error)) string {
	t.Helper()

	announcement, err := told(at)
	require.NoError(t, err)
	return string(announcement.Payload())
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

func TestARoundNobodyHeldGroundInHasAnEmptyPodium(t *testing.T) {
	assert.JSONEq(t, `{"number":1,"podium":[]}`, payloadOf(t, announcements.DayOf(0, 1).Placed(nil).Announcement))
}

var at = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)

func TestCountriesLevelAtTheTopAreChampionsTogether(t *testing.T) {
	announcement, err := announcements.ChampionOf(2, []announcements.Place{
		announcements.PlaceOf("de", 1, 100),
		announcements.PlaceOf("fr", 1, 100),
		announcements.PlaceOf("it", 3, 40),
	}).Announcement(at)

	require.NoError(t, err)
	assert.JSONEq(t, `{"season":2,"countries":["de","fr"]}`, string(announcement.Payload()))
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
		"bomb 7":         announcements.BombOf(7, "fr", "de", 7, 42, 3).Announcement,
		"bomb 8":         announcements.BombOf(8, "fr", "de", 7, 42, 3).Announcement,
		"fortify 7":      announcements.FortifyOf(7, "fr", "is", 3, 60).Announcement,
		"knock-out 7":    announcements.KnockOutOf(7, "fr", "de").Announcement,
		"mute 1":         announcements.MutedOf(mutes.MuteID{15: 1}, "guest_a1b2c3", mutes.Chat, time.Hour).Announcement,
		"day 5":          announcements.DayOf(0, 5).Announcement,
		"day 5 of 1":     announcements.DayOf(1, 5).Announcement,
		"battle 5":       announcements.BattleOf(0, 5, false, 2).Announcement,
		"champion of 0":  announcements.ChampionOf(0, nil).Announcement,
		"flag mission":   announcements.FlagMissionOf("f1", "fr", 6).Announcement,
		"flag mission 2": announcements.FlagMissionOf("f2", "fr", 6).Announcement,
	} {
		first, err := told(at)
		require.NoError(t, err)
		again, err := told(at.Add(time.Hour))
		require.NoError(t, err)

		assert.Equal(t, first.ID(), again.ID(), "%s told twice", name)
		ids[first.ID()] += name
	}

	assert.Len(t, ids, 11, "each fact has an id no other fact has")
}

func TestAFlagMissionIsToldInItsFlagsRoomAndTheWorld(t *testing.T) {
	announcement, err := announcements.FlagMissionOf("f1", "fr", 6).Announcement(at)
	require.NoError(t, err)

	assert.Equal(t, announcements.KindFlagMission, announcement.Kind())
	assert.True(t, announcement.Kind().Known())
	assert.Equal(t, messages.FlagRoom("fr"), announcement.Room())
	assert.True(t, announcement.In(messages.FlagRoom("fr")))
	assert.True(t, announcement.In(messages.World))
	assert.False(t, announcement.In(messages.FlagRoom("de")))
}

func TestEveryOtherAnnouncementIsTheWorldsAlone(t *testing.T) {
	announcement, err := announcements.BombOf(7, "fr", "de", 7, 42, 3).Announcement(at)
	require.NoError(t, err)

	assert.Equal(t, messages.World, announcement.Room())
	assert.False(t, announcement.In(messages.FlagRoom("fr")), "a bomb by France is not France's room's news")
}

func TestAFlagMissionWithNoIDIsRefused(t *testing.T) {
	_, err := announcements.FlagMissionIDOf("")

	assert.ErrorIs(t, err, announcements.ErrNoFlagMission)
}
