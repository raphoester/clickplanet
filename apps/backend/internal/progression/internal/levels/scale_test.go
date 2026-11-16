package levels_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
)

var (
	member = accounts.AccountOf(true, time.Time{})
	guest  = accounts.AccountOf(false, time.Time{})
)

// A regular player's first day: each count differs from the others, so a source that reads the wrong one shows.
var firstDay = stats.StatsSnapshot{
	TilesTaken:     600,
	DaysPlayed:     1,
	BombsDropped:   3,
	BonusesClaimed: 4,
	RightAnswers:   2,
	TilesFortified: 1_234,
	MessagesSent:   40,
	MissionsByKind: map[stats.MissionKind]stats.MissionTally{
		"take_for": {Done: 1, Goals: 180}, "catch_boxes": {Done: 1, Goals: 4}, "answer_quizzes": {Done: 1, Goals: 2},
	},
}.Restore()

func TestEachSourceGivesItsOwnXP(t *testing.T) {
	for _, c := range []struct {
		name   string
		source levels.Source
		xp     levels.XP
	}{
		{name: "a tile taken is 1", source: levels.TilesTaken{}, xp: 600},
		{name: "the first take of a UTC day is 100", source: levels.DaysPlayed{}, xp: 100},
		{name: "a mission is its goal times its kind's rate", source: levels.MissionsDone{}, xp: 180 + 4*50 + 2*80},
		{name: "a bomb dropped is 50", source: levels.BombsDropped{}, xp: 150},
		{name: "a box caught is 30", source: levels.BoxesCaught{}, xp: 120},
		{name: "a right answer is 30", source: levels.RightAnswers{}, xp: 60},
		{name: "a tile fortified is 0.2, floored", source: levels.TilesFortified{}, xp: 246},
		{name: "a message is nothing", source: levels.MessagesSent{}, xp: 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.xp, c.source.XP(firstDay))
		})
	}
}

func TestEachMissionKindPaysItsGoalTimesItsRate(t *testing.T) {
	for _, c := range []struct {
		rate  levels.MissionRate
		goals uint64
		xp    levels.XP
	}{
		{rate: levels.TakeFor{}, goals: 240, xp: 240},
		{rate: levels.TakeFrom{}, goals: 160, xp: 240},
		{rate: levels.TakeFrom{}, goals: 81, xp: 121},
		{rate: levels.TakeIn{}, goals: 600, xp: 600},
		{rate: levels.TakeFromIn{}, goals: 80, xp: 240},
		{rate: levels.CatchBoxes{}, goals: 6, xp: 300},
		{rate: levels.AnswerQuizzes{}, goals: 4, xp: 320},
		{rate: levels.ShieldTiles{}, goals: 3, xp: 240},
		{rate: levels.FortifyTiles{}, goals: 60, xp: 240},
		{rate: levels.BombIn{}, goals: 2, xp: 300},
	} {
		t.Run(string(c.rate.Kind()), func(t *testing.T) {
			assert.Equal(t, c.xp, c.rate.XP(c.goals))
		})
	}
}

func TestEveryKindOfMissionHasOneRate(t *testing.T) {
	var told []levels.MissionKind
	for number, name := range missionsv1.MissionKind_name {
		if number != int32(missionsv1.MissionKind_MISSION_KIND_UNSPECIFIED) {
			told = append(told, levels.MissionKind(strings.ToLower(strings.TrimPrefix(name, "MISSION_KIND_"))))
		}
	}

	rated := make([]levels.MissionKind, 0, len(told))
	for _, rate := range (levels.MissionsDone{}).Rates() {
		rated = append(rated, rate.Kind())
	}

	assert.ElementsMatch(t, told, rated)
}

func TestTheLevelIsTheSumOfEverySourceOnTheCurve(t *testing.T) {
	standing := levels.Scale{}.StandingOf(member, firstDay)

	assert.Equal(t, levels.XP(600+100+540+150+120+60+246), standing.XP())
	assert.Equal(t, levels.Level(6), standing.Level())
	assert.Equal(t, levels.XP(1_750), standing.LevelXP())
	assert.Equal(t, levels.XP(3_024), standing.NextLevelXP())
}

func TestAGuestsRecordReachesTheLevelItWouldHaveOnceLinked(t *testing.T) {
	assert.Equal(t, levels.XP(1_816), levels.Scale{}.XPOf(firstDay))
	assert.Equal(t, levels.Level(6), levels.Scale{}.Reach(firstDay))
	assert.Equal(t, levels.Level(1), levels.Scale{}.Reach(stats.Stats{}))
}

func TestAGuestHasNoXPAndNoLevel(t *testing.T) {
	for _, c := range []struct {
		name    string
		account accounts.Account
		record  levels.Record
		empty   bool
		level   levels.Level
		xp      levels.XP
	}{
		{name: "a guest that did nothing", account: guest, record: stats.Stats{}, empty: true},
		{name: "a guest that played a whole day", account: guest, record: firstDay, empty: true},
		{name: "a player that did nothing is at level 1", account: member, record: stats.Stats{}, level: 1},
		{name: "a player that played a whole day", account: member, record: firstDay, level: 6, xp: 1_816},
	} {
		t.Run(c.name, func(t *testing.T) {
			standing := levels.Scale{}.StandingOf(c.account, c.record)

			assert.Equal(t, c.empty, standing.Empty())
			assert.Equal(t, c.level, standing.Level())
			assert.Equal(t, c.xp, standing.XP())
		})
	}
}
