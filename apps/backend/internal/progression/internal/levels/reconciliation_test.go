package levels_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
)

func took(account byte, tiles uint64) stats.Stats {
	return stats.StatsSnapshot{Account: accounts.AccountID{15: account}, TilesTaken: tiles}.Restore()
}

var (
	page = []stats.Stats{
		took(1, 112),
		took(2, 112),
		took(3, 112),
		took(4, 112),
		took(5, 0),
		took(6, 5_000),
	}
	known = map[accounts.AccountID]accounts.Account{
		{15: 1}: member, {15: 2}: member, {15: 3}: member, {15: 4}: member, {15: 5}: guest,
	}
	kept = levels.ByAccount{{15: 1}: 2, {15: 2}: 3, {15: 3}: 7, {15: 5}: 4, {15: 6}: 1}
)

func TestWhileHiddenEachLinkedAccountGetsTheExactLevelItsStatsReachAndAGuestNone(t *testing.T) {
	reconciliation := levels.Scale{}.ReconciliationOf(page, known, kept, levels.Hidden)

	assert.Equal(t, levels.ByAccount{{15: 1}: 3, {15: 4}: 3}, reconciliation.Raised(), "one with no level is raised too")
	assert.Equal(t, levels.ByAccount{{15: 3}: 3}, reconciliation.Lowered())
	assert.ElementsMatch(t, []accounts.AccountID{{15: 5}, {15: 6}}, reconciliation.Removed(),
		"a guest, and an account player does not know")
}

func TestOnceShownALevelOnlyGoesUp(t *testing.T) {
	reconciliation := levels.Scale{}.ReconciliationOf(page, known, kept, levels.Shown)

	assert.Equal(t, levels.ByAccount{{15: 1}: 3, {15: 4}: 3}, reconciliation.Raised())
	assert.Empty(t, reconciliation.Lowered())
	assert.Empty(t, reconciliation.Removed())
}

func TestAGuestWithNoLevelIsLeftAlone(t *testing.T) {
	reconciliation := levels.Scale{}.ReconciliationOf([]stats.Stats{took(5, 5_000)}, known, levels.ByAccount{}, levels.Hidden)

	assert.Empty(t, reconciliation.Raised())
	assert.Empty(t, reconciliation.Lowered())
	assert.Empty(t, reconciliation.Removed())
}

func TestALevelUpIsSilentWhileThePlayersSeeNoLevel(t *testing.T) {
	assert.True(t, levels.Hidden.Silent())
	assert.False(t, levels.Shown.Silent())
}
