package players_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

func tiles(n uint64) players.Career { return players.Career{Stats: players.Stats{TilesTaken: n}} }

func streak(days uint32) players.Career {
	return players.Career{Stats: players.Stats{StreakBest: days}}
}

func TestEachTitleIsEarnedAtItsThresholdAndNotBefore(t *testing.T) {
	for _, c := range []struct {
		title         players.Title
		before, after players.Career
	}{
		{players.Settler{}, tiles(99), tiles(100)},
		{players.Governor{}, tiles(999), tiles(1_000)},
		{players.Conqueror{}, tiles(9_999), tiles(10_000)},
		{players.Emperor{}, tiles(99_999), tiles(100_000)},
		{players.Loyal{}, streak(6), streak(7)},
		{players.Devoted{}, streak(29), streak(30)},
		{players.Unbroken{}, streak(99), streak(100)},
	} {
		assert.False(t, c.title.EarnedBy(c.before), "%s at %+v", c.title.ID(), c.before)
		assert.True(t, c.title.EarnedBy(c.after), "%s at %+v", c.title.ID(), c.after)
	}
}

func TestAStreakTitleCountsTheBestStreakNotTheCurrentOne(t *testing.T) {
	assert.False(t, players.Devoted{}.EarnedBy(players.Career{Stats: players.Stats{StreakCurrent: 30, StreakBest: 1}}))
	assert.True(t, players.Devoted{}.EarnedBy(players.Career{Stats: players.Stats{StreakCurrent: 0, StreakBest: 30}}))
}

func TestAnAccountMadeBeforeNovemberIsOG(t *testing.T) {
	november := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

	assert.True(t, players.OG{}.EarnedBy(players.Career{CreatedAt: november.Add(-time.Millisecond)}))
	assert.False(t, players.OG{}.EarnedBy(players.Career{CreatedAt: november}))
	assert.False(t, players.OG{}.EarnedBy(players.Career{}), "an account auth does not know has no date")
}

func TestEveryTitleHasItsOwnIDTheStoreTakesAndAName(t *testing.T) {
	pattern := regexp.MustCompile(`^[a-z_]+$`)
	seen := cpcolls.NewSet[players.TitleID]()

	for _, title := range players.NewCatalog() {
		assert.Regexp(t, pattern, string(title.ID()))
		assert.NotEmpty(t, title.Name(), title.ID())
		assert.False(t, seen.Contains(title.ID()), "%s is listed twice", title.ID())
		seen.Add(title.ID())
	}
}
