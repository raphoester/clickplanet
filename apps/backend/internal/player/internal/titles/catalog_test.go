package titles_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var member = players.AccountOf(true, time.Time{})

func careerOf(stats players.StatsSnapshot) titles.Career {
	return titles.CareerOf(stats.Restore(), member, seasons.Record{})
}

func tiles(n uint64) titles.Career { return careerOf(players.StatsSnapshot{TilesTaken: n}) }

func streak(days uint32) titles.Career { return careerOf(players.StatsSnapshot{StreakBest: days}) }

func messages(n uint64) titles.Career { return careerOf(players.StatsSnapshot{MessagesSent: n}) }

func fortified(n uint64) titles.Career { return careerOf(players.StatsSnapshot{TilesFortified: n}) }

func bombs(n uint64) titles.Career { return careerOf(players.StatsSnapshot{BombsDropped: n}) }

func bonuses(n uint64) titles.Career { return careerOf(players.StatsSnapshot{BonusesClaimed: n}) }

func answers(n uint64) titles.Career { return careerOf(players.StatsSnapshot{RightAnswers: n}) }

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
		{titles.Talker{}, messages(9), messages(10)},
		{titles.Chatterbox{}, messages(29), messages(30)},
		{titles.Socialite{}, messages(99), messages(100)},
		{titles.Icon{}, messages(999), messages(1_000)},
		{titles.Guard{}, fortified(99), fortified(100)},
		{titles.Defender{}, fortified(999), fortified(1_000)},
		{titles.Protector{}, fortified(9_999), fortified(10_000)},
		{titles.Warden{}, fortified(99_999), fortified(100_000)},
		{titles.Bomber{}, bombs(0), bombs(1)},
		{titles.Blaster{}, bombs(9), bombs(10)},
		{titles.Wrecker{}, bombs(49), bombs(50)},
		{titles.Destroyer{}, bombs(249), bombs(250)},
		{titles.Looter{}, bonuses(9), bonuses(10)},
		{titles.Hunter{}, bonuses(99), bonuses(100)},
		{titles.Pirate{}, bonuses(499), bonuses(500)},
		{titles.Captain{}, bonuses(2_499), bonuses(2_500)},
		{titles.Student{}, answers(9), answers(10)},
		{titles.Expert{}, answers(49), answers(50)},
		{titles.Professor{}, answers(249), answers(250)},
		{titles.Genius{}, answers(999), answers(1_000)},
	} {
		assert.False(t, c.title.EarnedBy(c.before), "%s at %+v", c.title.ID(), c.before)
		assert.True(t, c.title.EarnedBy(c.after), "%s at %+v", c.title.ID(), c.after)
	}
}

func TestAStreakTitleCountsTheBestStreakNotTheCurrentOne(t *testing.T) {
	assert.False(t, titles.Devoted{}.EarnedBy(careerOf(players.StatsSnapshot{Streak: players.StreakOf(30, players.Day{}), StreakBest: 1})))
	assert.True(t, titles.Devoted{}.EarnedBy(careerOf(players.StatsSnapshot{StreakBest: 30})))
}

func TestAnAccountMadeBeforeNovemberIsOG(t *testing.T) {
	november := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

	assert.True(t, titles.OG{}.EarnedBy(titles.CareerOf(players.Stats{}, players.AccountOf(true, november.Add(-time.Millisecond)), seasons.Record{})))
	assert.False(t, titles.OG{}.EarnedBy(titles.CareerOf(players.Stats{}, players.AccountOf(true, november), seasons.Record{})))
	assert.False(t, titles.OG{}.EarnedBy(titles.Career{}), "an account auth does not know has no date")
}

func TestASeasonPlayedMakesAVeteranAndItsFlagWinningAChampion(t *testing.T) {
	played := func(season seasons.Season) titles.Career {
		return titles.CareerOf(players.Stats{}, member, seasons.NewRecord(players.AccountID{}).With(season))
	}
	won, lost, later := played(seasons.Season{Number: 0, Flag: "fr", Won: true}), played(seasons.Season{Number: 0, Flag: "de"}),
		played(seasons.Season{Number: 1, Flag: "fr", Won: true})

	assert.True(t, titles.Veteran{Season: 0}.EarnedBy(won))
	assert.True(t, titles.Veteran{Season: 0}.EarnedBy(lost))
	assert.False(t, titles.Veteran{Season: 0}.EarnedBy(later))
	assert.False(t, titles.Veteran{Season: 0}.EarnedBy(tiles(1_000)), "tiles taken outside a season told")
	assert.True(t, titles.Champion{Season: 0}.EarnedBy(won))
	assert.False(t, titles.Champion{Season: 0}.EarnedBy(lost))
	assert.False(t, titles.Champion{Season: 0}.EarnedBy(later))
}

func TestEachSeasonTitleNamesItsSeason(t *testing.T) {
	assert.Equal(t, titles.ID("veteran_s0"), titles.Veteran{Season: 0}.ID())
	assert.Equal(t, "Veteran · S0", titles.Veteran{Season: 0}.Name())
	assert.Equal(t, titles.ID("champion_s1"), titles.Champion{Season: 1}.ID())
	assert.Equal(t, "Champion · S1", titles.Champion{Season: 1}.Name())
}

func TestEachTrackClimbsItsRanksByItsOwnMeasure(t *testing.T) {
	for _, track := range []titles.Track{
		titles.Conquest{},
		titles.Devotion{},
		titles.Chatter{},
		titles.Defense{},
		titles.Bombs{},
		titles.Loot{},
		titles.Quiz{},
	} {
		var previous uint64
		for _, rank := range track.Ranks() {
			assert.Greater(t, rank.Threshold(), previous, "%s: %s", track.ID(), rank.ID())
			n := rank.Threshold()
			reached := players.StatsSnapshot{
				TilesTaken: n, StreakBest: uint32(n), MessagesSent: n,
				BombsDropped: n, TilesFortified: n, BonusesClaimed: n, RightAnswers: n,
			}.Restore()
			assert.True(t, rank.EarnedBy(titles.CareerOf(reached, players.Account{}, seasons.Record{})),
				"%s: %s is earned at its threshold", track.ID(), rank.ID())
			previous = rank.Threshold()
		}
	}

	career := titles.CareerOf(players.StatsSnapshot{
		TilesTaken: 14_468, Streak: players.StreakOf(4, players.Day{}), StreakBest: 7, MessagesSent: 312,
		TilesFortified: 2_046, BombsDropped: 9, BonusesClaimed: 61, RightAnswers: 23,
	}.Restore(), players.Account{}, seasons.Record{})
	assert.Equal(t, uint64(14_468), titles.Conquest{}.Progress(career))
	assert.Equal(t, uint64(4), titles.Devotion{}.Progress(career), "the streak a player is on now, not its best")
	assert.Equal(t, uint64(312), titles.Chatter{}.Progress(career))
	assert.Equal(t, uint64(2_046), titles.Defense{}.Progress(career))
	assert.Equal(t, uint64(9), titles.Bombs{}.Progress(career))
	assert.Equal(t, uint64(61), titles.Loot{}.Progress(career))
	assert.Equal(t, uint64(23), titles.Quiz{}.Progress(career))
}

func TestEveryTitleHasItsOwnIDTheStoreTakesAndAName(t *testing.T) {
	pattern := regexp.MustCompile(`^[a-z0-9_]+$`)
	seen := cpcolls.NewSet[titles.ID]()

	for _, title := range titles.NewCatalog().Titles() {
		assert.Regexp(t, pattern, string(title.ID()))
		assert.NotEmpty(t, title.Name(), title.ID())
		assert.False(t, seen.Contains(title.ID()), "%s is listed twice", title.ID())
		seen.Add(title.ID())
	}
}
