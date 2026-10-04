package lead_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/lead"
)

var start = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func shares(tiles map[string]uint32) lead.Shares { return lead.SharesOf(tiles) }

func TestTheLeaderHoldsTheMostTiles(t *testing.T) {
	leader, ok := shares(map[string]uint32{"fr": 400, "dz": 500, "bg": 100}).Leader()

	require.True(t, ok)
	assert.Equal(t, "dz", leader)
}

func TestATieGoesToTheFirstCode(t *testing.T) {
	leader, ok := shares(map[string]uint32{"fr": 400, "dz": 400}).Leader()

	require.True(t, ok)
	assert.Equal(t, "dz", leader)
}

func TestAnEmptyMapHasNoLeader(t *testing.T) {
	_, ok := shares(map[string]uint32{"fr": 0}).Leader()
	assert.False(t, ok)

	_, ok = shares(map[string]uint32{}).Leader()
	assert.False(t, ok)
}

func TestTheFirstReadingNamesTheLeaderWithoutAPass(t *testing.T) {
	race, _, passed := lead.NewRace(lead.Config{}).Next(shares(map[string]uint32{"fr": 500, "dz": 400}), start)

	assert.False(t, passed, "a restart in the middle of the finale says nothing")
	assert.Equal(t, "fr", race.Leader())
}

func TestAChallengerThatHoldsItsMarginForTheHoldPasses(t *testing.T) {
	race := lead.NewRace(lead.Config{Margin: 50, Hold: 30 * time.Second})
	race, _, _ = race.Next(shares(map[string]uint32{"fr": 500, "bg": 400}), start)

	race, _, passed := race.Next(shares(map[string]uint32{"fr": 500, "bg": 560}), start.Add(time.Second))
	require.False(t, passed, "not yet held")

	race, _, passed = race.Next(shares(map[string]uint32{"fr": 500, "bg": 570}), start.Add(20*time.Second))
	require.False(t, passed)

	race, pass, passed := race.Next(shares(map[string]uint32{"fr": 500, "bg": 580}), start.Add(31*time.Second))
	require.True(t, passed)
	assert.Equal(t, "bg", pass.Leader())
	assert.Equal(t, "fr", pass.Passed())
	assert.Equal(t, "bg", race.Leader())

	_, _, passed = race.Next(shares(map[string]uint32{"fr": 500, "bg": 580}), start.Add(time.Minute))
	assert.False(t, passed, "a lead is told once")
}

func TestALeadUnderTheMarginIsNoPassHoweverLongItLasts(t *testing.T) {
	race := lead.NewRace(lead.Config{Margin: 50, Hold: 30 * time.Second})
	race, _, _ = race.Next(shares(map[string]uint32{"fr": 500, "bg": 400}), start)

	for minute := range 10 {
		var passed bool
		race, _, passed = race.Next(shares(map[string]uint32{"fr": 500, "bg": 549}), start.Add(time.Duration(minute+1)*time.Minute))
		require.False(t, passed)
	}
	assert.Equal(t, "fr", race.Leader())
}

func TestAFlappingLeadSaysNothing(t *testing.T) {
	race := lead.NewRace(lead.Config{Margin: 50, Hold: 30 * time.Second})
	race, _, _ = race.Next(shares(map[string]uint32{"fr": 500, "bg": 400}), start)

	for second := 1; second <= 600; second++ {
		bg := uint32(400)
		if (second/20)%2 == 1 {
			bg = 600
		}
		var passed bool
		race, _, passed = race.Next(shares(map[string]uint32{"fr": 500, "bg": bg}), start.Add(time.Duration(second)*time.Second))
		require.False(t, passed, "second %d", second)
	}
}

func TestAChallengerOvertakenRestartsTheHold(t *testing.T) {
	race := lead.NewRace(lead.Config{Margin: 50, Hold: 30 * time.Second})
	race, _, _ = race.Next(shares(map[string]uint32{"fr": 500, "bg": 400, "dz": 400}), start)

	race, _, _ = race.Next(shares(map[string]uint32{"fr": 500, "bg": 600, "dz": 400}), start.Add(time.Second))
	race, _, passed := race.Next(shares(map[string]uint32{"fr": 500, "bg": 600, "dz": 700}), start.Add(20*time.Second))
	require.False(t, passed)

	race, _, passed = race.Next(shares(map[string]uint32{"fr": 500, "bg": 600, "dz": 700}), start.Add(40*time.Second))
	require.False(t, passed, "dz has led for 20s, not 30s")

	_, pass, passed := race.Next(shares(map[string]uint32{"fr": 500, "bg": 600, "dz": 700}), start.Add(51*time.Second))
	require.True(t, passed)
	assert.Equal(t, "dz", pass.Leader())
	assert.Equal(t, "fr", pass.Passed())
}

func TestAChallengerIsTimedFromTheFirstReadingThatSawItLead(t *testing.T) {
	race := lead.NewRace(lead.Config{Margin: 1, Hold: time.Second})
	race, _, _ = race.Next(shares(map[string]uint32{"fr": 500, "bg": 400}), start)

	race, _, passed := race.Next(shares(map[string]uint32{"fr": 500, "bg": 501}), start.Add(time.Minute))
	require.False(t, passed, "a reading a minute after the last is still the first to see bg ahead")

	_, pass, passed := race.Next(shares(map[string]uint32{"fr": 500, "bg": 501}), start.Add(time.Minute+time.Second))
	require.True(t, passed)
	assert.Equal(t, "bg", pass.Leader())
}
