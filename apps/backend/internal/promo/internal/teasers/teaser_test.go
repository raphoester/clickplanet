package teasers_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
)

var start = time.Date(2026, 11, 14, 20, 0, 0, 0, time.UTC)

func ringWar() teasers.Teaser {
	return teasers.Teaser{
		Battle: "entry-7", StartsAt: start, EndsAt: start.Add(time.Hour), Description: "Close a ring around any land.",
		Times: 3, Cards: []teasers.Card{
			{BackgroundURL: "https://media.test/ring.jpg", Headline: "Circle any land with your flag", Line: "Every click can close a ring."},
			{BackgroundURL: "https://media.test/points.jpg", Headline: "Gain the most land"},
		},
	}
}

func TestATeaserGoesOutFromADayAndAHalfToAnHourBeforeItsBattle(t *testing.T) {
	notBefore, by := ringWar().Window()

	assert.Equal(t, time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC), notBefore)
	assert.Equal(t, time.Date(2026, 11, 14, 19, 0, 0, 0, time.UTC), by)
}

func TestTheBattlesDueAreThoseNoPostTeasesInTheirOrder(t *testing.T) {
	first, second, third := ringWar(), ringWar(), ringWar()
	second.Battle, third.Battle = "season-1/finale", "entry-9"

	due := teasers.Due([]teasers.Teaser{first, second, third}, []teasers.BattleID{"entry-9", "entry-1"})

	assert.Equal(t, []teasers.Teaser{first, second}, due)
	assert.Empty(t, teasers.Due(nil, nil))
}

func TestThePostsReadForWhatWasTeasedReachTwoLeadsBack(t *testing.T) {
	now := time.Date(2026, 11, 13, 7, 43, 0, 0, time.UTC)

	assert.Equal(t, now.Add(-60*time.Hour), teasers.TeasedSince(now))
}

func TestAFolderIsNamedByItsBattleWithNoMarkAPathReads(t *testing.T) {
	for id, name := range map[teasers.BattleID]string{
		"0b8e5c2a-3f2e-4c1a-9a51-7f2d3c4b5a69": "0b8e5c2a-3f2e-4c1a-9a51-7f2d3c4b5a69",
		"season-1/finale":                      "season-1_finale",
		"../../etc":                            ".._.._etc",
		"..":                                   "_..",
		"":                                     "_",
	} {
		assert.Equal(t, name, teasers.Teaser{Battle: id}.FolderName(), id)
	}
}
