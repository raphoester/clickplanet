package titles_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var member = players.Account{Linked: true}

func tiles(n uint64) titles.Career {
	return titles.Career{Stats: players.Stats{TilesTaken: n}, Account: member}
}

func streak(days uint32) titles.Career {
	return titles.Career{Stats: players.Stats{StreakBest: days}, Account: member}
}

func messages(n uint64) titles.Career {
	return titles.Career{Stats: players.Stats{MessagesSent: n}, Account: member}
}

func TestEachTitleIsEarnedAtItsThresholdAndNotBefore(t *testing.T) {
	for _, c := range []struct {
		title         titles.Title
		before, after titles.Career
	}{
		{titles.Settler{}, tiles(99), tiles(100)},
		{titles.Raider{}, tiles(999), tiles(1_000)},
		{titles.Warlord{}, tiles(9_999), tiles(10_000)},
		{titles.Conqueror{}, tiles(99_999), tiles(100_000)},
		{titles.Warmaster{}, tiles(999_999), tiles(1_000_000)},
		{titles.Loyal{}, streak(6), streak(7)},
		{titles.Devoted{}, streak(29), streak(30)},
		{titles.Unbroken{}, streak(99), streak(100)},
		{titles.Talker{}, messages(99), messages(100)},
		{titles.Chatterbox{}, messages(999), messages(1_000)},
		{titles.Socialite{}, messages(9_999), messages(10_000)},
		{titles.Icon{}, messages(99_999), messages(100_000)},
	} {
		assert.False(t, c.title.EarnedBy(c.before), "%s at %+v", c.title.ID(), c.before)
		assert.True(t, c.title.EarnedBy(c.after), "%s at %+v", c.title.ID(), c.after)
	}
}

func TestAStreakTitleCountsTheBestStreakNotTheCurrentOne(t *testing.T) {
	assert.False(t, titles.Devoted{}.EarnedBy(titles.Career{Stats: players.Stats{StreakCurrent: 30, StreakBest: 1}}))
	assert.True(t, titles.Devoted{}.EarnedBy(titles.Career{Stats: players.Stats{StreakCurrent: 0, StreakBest: 30}}))
}

func TestAnAccountMadeBeforeNovemberIsOG(t *testing.T) {
	november := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

	assert.True(t, titles.OG{}.EarnedBy(titles.Career{Account: players.Account{Linked: true, CreatedAt: november.Add(-time.Millisecond)}}))
	assert.False(t, titles.OG{}.EarnedBy(titles.Career{Account: players.Account{Linked: true, CreatedAt: november}}))
	assert.False(t, titles.OG{}.EarnedBy(titles.Career{}), "an account auth does not know has no date")
}

func TestEachTrackClimbsItsRanksByItsOwnMeasure(t *testing.T) {
	for _, track := range []titles.Track{titles.Conquest{}, titles.Devotion{}, titles.Chatter{}} {
		var previous uint64
		for _, rank := range track.Ranks() {
			assert.Greater(t, rank.Threshold(), previous, "%s: %s", track.ID(), rank.ID())
			reached := players.Stats{TilesTaken: rank.Threshold(), StreakBest: uint32(rank.Threshold()), MessagesSent: rank.Threshold()}
			assert.True(t, rank.EarnedBy(titles.Career{Stats: reached}),
				"%s: %s is earned at its threshold", track.ID(), rank.ID())
			previous = rank.Threshold()
		}
	}

	career := titles.Career{Stats: players.Stats{TilesTaken: 14_468, StreakCurrent: 4, StreakBest: 7, MessagesSent: 312}}
	assert.Equal(t, uint64(14_468), titles.Conquest{}.Progress(career))
	assert.Equal(t, uint64(4), titles.Devotion{}.Progress(career), "the streak a player is on now, not its best")
	assert.Equal(t, uint64(312), titles.Chatter{}.Progress(career))
}

func TestEveryTitleHasItsOwnIDTheStoreTakesAndAName(t *testing.T) {
	pattern := regexp.MustCompile(`^[a-z_]+$`)
	seen := cpcolls.NewSet[titles.ID]()

	for _, title := range titles.NewCatalog().Titles() {
		assert.Regexp(t, pattern, string(title.ID()))
		assert.NotEmpty(t, title.Name(), title.ID())
		assert.False(t, seen.Contains(title.ID()), "%s is listed twice", title.ID())
		seen.Add(title.ID())
	}
}
