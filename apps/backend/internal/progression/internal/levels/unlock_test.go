package levels_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

func TestEachKindIsUnlockedFromItsLevelOn(t *testing.T) {
	for level, kinds := range map[levels.Level][]levels.Kind{
		0:  {},
		1:  {},
		2:  {"spread_clicks"},
		3:  {"spread_clicks", "shields"},
		4:  {"spread_clicks", "shields"},
		5:  {"spread_clicks", "shields", "enclose_clicks"},
		6:  {"spread_clicks", "shields", "enclose_clicks"},
		7:  {"spread_clicks", "shields", "enclose_clicks", "bomb"},
		9:  {"spread_clicks", "shields", "enclose_clicks", "bomb"},
		10: {"spread_clicks", "shields", "enclose_clicks", "bomb", "virus"},
		12: {"spread_clicks", "shields", "enclose_clicks", "bomb", "virus"},
		13: {"spread_clicks", "shields", "enclose_clicks", "bomb", "virus", "mine"},
		14: {"spread_clicks", "shields", "enclose_clicks", "bomb", "virus", "mine"},
		15: {"spread_clicks", "shields", "enclose_clicks", "bomb", "virus", "mine", "rally"},
		19: {"spread_clicks", "shields", "enclose_clicks", "bomb", "virus", "mine", "rally"},
		20: {"spread_clicks", "shields", "enclose_clicks", "bomb", "virus", "mine", "rally", "trap"},
		40: {"spread_clicks", "shields", "enclose_clicks", "bomb", "virus", "mine", "rally", "trap"},
	} {
		assert.Equal(t, kinds, levels.Ladder{}.UnlockedAt(level), "level %d", level)
	}
}

func TestEachKindIsOnTheLadderOnce(t *testing.T) {
	seen := cpcolls.NewSet[levels.Kind]()
	for _, unlock := range (levels.Ladder{}).Unlocks() {
		assert.False(t, seen.Contains(unlock.Kind()), "%s", unlock.Kind())
		seen.Add(unlock.Kind())
		assert.Greater(t, unlock.Level(), levels.Level(1), "a level-1 kind is the bonuses' open kinds")
	}
}
