package rules_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

func TestTheSeededSetPlays(t *testing.T) {
	set, err := rules.Seeded().Restore()

	require.NoError(t, err)
	assert.Equal(t, rules.Seeded(), set.TakeSnapshot())
}

func TestASetIsRefusedWhenAnyValueCannotBeJudgedWith(t *testing.T) {
	for reason, change := range map[string]func(*rules.Snapshot){
		"a name is a lowercase letter":             func(s *rules.Snapshot) { s.Name = "Rush" },
		"bans.banDurations is empty":               func(s *rules.Snapshot) { s.Bans.BanDurations = nil },
		"bans.banDurations[1] is -1h0m0s":          func(s *rules.Snapshot) { s.Bans.BanDurations[1] = -time.Hour },
		"bans.reflagInterval is 0s":                func(s *rules.Snapshot) { s.Bans.ReflagInterval = 0 },
		"jury.minSuspects is 0":                    func(s *rules.Snapshot) { s.Jury.MinSuspects = 0 },
		"retaker.minReactions is 0":                func(s *rules.Snapshot) { s.Retaker.MinReactions = 0 },
		"sequencer.certainSteps is 10, under":      func(s *rules.Snapshot) { s.Sequencer.CertainSteps = 10 },
		"sequencer.minShare is NaN":                func(s *rules.Snapshot) { s.Sequencer.MinShare = math.NaN() },
		"metronome.shape.clicks is 5000, over":     func(s *rules.Snapshot) { s.Metronome.Shape.Clicks = 5000 },
		"metronome.clock.minCoherence is 2":        func(s *rules.Snapshot) { two := 2.0; s.Metronome.Clock.MinCoherence = &two },
		"metronome.stamina.window is 0s":           func(s *rules.Snapshot) { s.Metronome.Stamina.Window = 0 },
		"defender.minShare is 1.5":                 func(s *rules.Snapshot) { s.Defender.MinShare = 1.5 },
		"catcher.certainMedian is 5s, over":        func(s *rules.Snapshot) { s.Catcher.CertainMedian = 5 * time.Second },
		"cohort.minMembers is 1, under":            func(s *rules.Snapshot) { s.Cohort.MinMembers = 1 },
		"cohort.certainMembers is 1, under":        func(s *rules.Snapshot) { s.Cohort.CertainMembers = 1 },
		"scraper.certainOffMap is 0":               func(s *rules.Snapshot) { s.Scraper.CertainOffMap = 0 },
		"churner.relay.minClicks is 1, under":      func(s *rules.Snapshot) { s.Churner.Relay.MinClicks = 1 },
		"hopper.minAngle is 270, over half a turn": func(s *rules.Snapshot) { s.Hopper.MinAngle = 270 },
		"rotator.maxShare is 0":                    func(s *rules.Snapshot) { s.Rotator.MaxShare = 0 },
	} {
		t.Run(reason, func(t *testing.T) {
			_, err := rules.Named("rush", change).Restore()

			require.ErrorIs(t, err, rules.ErrInvalidSet)
			assert.Contains(t, err.Error(), reason)
		})
	}
}

func TestARefusalSaysEveryValueItRefuses(t *testing.T) {
	_, err := rules.Named("rush", func(s *rules.Snapshot) {
		s.Retaker.MinReactions = 0
		s.Rotator.MinSeconds = 0
	}).Restore()

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "retaker.minReactions is 0, and must be above 0; rotator.minSeconds is 0")
}

func TestAZeroThatNeverReadsIsARuleOfItsOwn(t *testing.T) {
	_, err := rules.Named("quiet", func(s *rules.Snapshot) {
		s.Retaker.MinTiles = 0
		s.Defender.MinShare = 0
		s.Rotator.MinSlots = 0
		s.Metronome.Stamina.MinBusy = 0
		s.Churner.V6, s.Churner.V4 = s.Churner.V4, s.Churner.V6
	}).Restore()

	assert.NoError(t, err)
}

func TestTwoSetsAreEqualWhenTheyJudgeAlike(t *testing.T) {
	coherence := func(v float64) *float64 { return &v }
	restore := func(change func(*rules.Snapshot)) rules.Set {
		t.Helper()
		set, err := rules.Named("rush", change).Restore()
		require.NoError(t, err)
		return set
	}

	same := restore(func(s *rules.Snapshot) { s.Metronome.Clock.MinCoherence = coherence(0.5) })

	assert.True(t, same.Equal(restore(func(s *rules.Snapshot) { s.Metronome.Clock.MinCoherence = coherence(0.5) })),
		"a bound behind a pointer is compared by its value")
	assert.False(t, same.Equal(restore(func(s *rules.Snapshot) { s.Metronome.Clock.MinCoherence = coherence(0.6) })))
	assert.False(t, same.Equal(restore(rules.AsIs)))
}
