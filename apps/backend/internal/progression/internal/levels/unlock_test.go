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
		3:  {},
		4:  {"enclose_clicks"},
		5:  {"enclose_clicks", "bomb"},
		6:  {"enclose_clicks", "bomb", "virus"},
		7:  {"enclose_clicks", "bomb", "virus"},
		8:  {"enclose_clicks", "bomb", "virus", "mine"},
		9:  {"enclose_clicks", "bomb", "virus", "mine"},
		10: {"enclose_clicks", "bomb", "virus", "mine", "rally"},
		40: {"enclose_clicks", "bomb", "virus", "mine", "rally"},
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
