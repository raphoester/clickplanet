package rules_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

func TestTheSeededSetPlays(t *testing.T) {
	set, err := rules.Seeded().Restore(guard.Parts(), rules.Seeded())

	require.NoError(t, err)
	assert.True(t, proto.Equal(rules.Seeded().Rules, set.TakeSnapshot().Rules))
}

func TestADefaultIsRefusedWhenAnyValueCannotBeJudgedWith(t *testing.T) {
	for reason, change := range map[string]func(*antibotv1.Rules){
		"bans.banDurations is empty":           func(r *antibotv1.Rules) { r.Bans.BanDurations = nil },
		"bans.banDurations[1] is -1h0m0s":      func(r *antibotv1.Rules) { r.Bans.BanDurations[1] = durationpb.New(-time.Hour) },
		"bans.reflagInterval is 0s":            func(r *antibotv1.Rules) { r.Bans.ReflagInterval = nil },
		"jury.minSuspects is 0":                func(r *antibotv1.Rules) { r.Jury.MinSuspects = proto.Uint32(0) },
		"retaker.minReactions is 0":            func(r *antibotv1.Rules) { r.Retaker.MinReactions = proto.Uint32(0) },
		"sequencer.certainSteps is 10, under":  func(r *antibotv1.Rules) { r.Sequencer.CertainSteps = proto.Uint32(10) },
		"sequencer.minShare is NaN":            func(r *antibotv1.Rules) { r.Sequencer.MinShare = proto.Float64(math.NaN()) },
		"metronome.shape.clicks is 5000, over": func(r *antibotv1.Rules) { r.Metronome.Shape.Clicks = proto.Uint32(5000) },
		"metronome.clock.minCoherence is 2":    func(r *antibotv1.Rules) { r.Metronome.Clock.MinCoherence = proto.Float64(2) },
		"metronome.stamina.window is 0s":       func(r *antibotv1.Rules) { r.Metronome.Stamina.Window = nil },
		"defender.minShare is 1.5":             func(r *antibotv1.Rules) { r.Defender.MinShare = proto.Float64(1.5) },
		"catcher.certainMedian is 5s, over":    func(r *antibotv1.Rules) { r.Catcher.CertainMedian = durationpb.New(5 * time.Second) },
		"cohort.minMembers is 1, under":        func(r *antibotv1.Rules) { r.Cohort.MinMembers = proto.Uint32(1) },
		"cohort.certainMembers is 1, under":    func(r *antibotv1.Rules) { r.Cohort.CertainMembers = proto.Uint32(1) },
		"scraper.certainOffMap is 0":           func(r *antibotv1.Rules) { r.Scraper.CertainOffMap = proto.Uint32(0) },
		"churner.relay.minClicks is 1, under":  func(r *antibotv1.Rules) { r.Churner.Relay.MinClicks = proto.Uint32(1) },
		"hopper.minAngle is 270, over":         func(r *antibotv1.Rules) { r.Hopper.MinAngle = proto.Float64(270) },
		"rotator.maxShare is 0":                func(r *antibotv1.Rules) { r.Rotator.MaxShare = proto.Float64(0) },
		"rotator.window is 0s":                 func(r *antibotv1.Rules) { r.Rotator = nil },
	} {
		t.Run(reason, func(t *testing.T) {
			_, err := rules.Named(rules.Default, change).Restore(guard.Parts(), rules.Seeded())

			require.ErrorIs(t, err, rules.ErrInvalidSet)
			assert.Contains(t, err.Error(), reason)
		})
	}
}

func TestANameOutsideThePatternIsRefused(t *testing.T) {
	_, err := rules.Named("Rush", rules.AsIs).Restore(guard.Parts(), rules.Seeded())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "a name is a lowercase letter")
}

func TestARefusalSaysEveryValueItRefuses(t *testing.T) {
	_, err := rules.Named("rush", func(r *antibotv1.Rules) {
		r.Retaker.MinReactions = proto.Uint32(0)
		r.Rotator.MinSeconds = proto.Uint32(0)
	}).Restore(guard.Parts(), rules.Seeded())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "retaker.minReactions is 0, and must be above 0; rotator.minSeconds is 0")
}

func TestASetJudgesWithDefaultsValueWhereItHoldsNone(t *testing.T) {
	crowd := rules.Snapshot{Name: "rush", Rules: &antibotv1.Rules{
		Cohort: &antibotv1.CohortRules{CertainMembers: proto.Uint32(50)},
		Bans:   &antibotv1.BanRules{Enforce: proto.Bool(false)},
	}}

	set, err := crowd.Restore(guard.Parts(), rules.Seeded())
	require.NoError(t, err)

	assert.True(t, proto.Equal(crowd.Rules, set.TakeSnapshot().Rules))
	assert.True(t, proto.Equal(rules.Named("rush", func(r *antibotv1.Rules) {
		r.Cohort.CertainMembers = proto.Uint32(50)
		r.Bans.Enforce = proto.Bool(false)
	}).Rules, set.TakePlayedSnapshot().Rules))
}

func TestASetIsCheckedOverDefault(t *testing.T) {
	_, err := rules.Snapshot{Name: "rush", Rules: &antibotv1.Rules{
		Cohort: &antibotv1.CohortRules{CertainMembers: proto.Uint32(1)},
	}}.Restore(guard.Parts(), rules.Seeded())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "cohort.certainMembers is 1, under")
}

func TestAZeroThatNeverReadsIsARuleOfItsOwn(t *testing.T) {
	_, err := rules.Named(rules.Default, func(r *antibotv1.Rules) {
		r.Retaker.MinTiles = proto.Uint32(0)
		r.Defender.MinShare = proto.Float64(0)
		r.Rotator.MinSlots = proto.Uint32(0)
		r.Metronome.Stamina.MinBusy = nil
		r.Churner.V6, r.Churner.V4 = nil, nil
		r.Hopper.Enabled = proto.Bool(false)
	}).Restore(guard.Parts(), rules.Seeded())

	assert.NoError(t, err)
}

func TestTwoSetsAreEqualWhenTheyJudgeAlike(t *testing.T) {
	restore := func(change func(*antibotv1.Rules)) rules.Set {
		t.Helper()
		set, err := rules.Named("rush", change).Restore(guard.Parts(), rules.Seeded())
		require.NoError(t, err)
		return set
	}

	same := restore(func(r *antibotv1.Rules) { r.Metronome.Clock.MinCoherence = proto.Float64(0.5) })

	assert.True(t, same.Equal(restore(func(r *antibotv1.Rules) { r.Metronome.Clock.MinCoherence = proto.Float64(0.5) })),
		"a bound behind a pointer is compared by its value")
	assert.False(t, same.Equal(restore(func(r *antibotv1.Rules) { r.Metronome.Clock.MinCoherence = proto.Float64(0.6) })))
	assert.False(t, same.Equal(restore(rules.AsIs)))

	sparse, err := rules.Snapshot{Name: "rush", Rules: &antibotv1.Rules{
		Metronome: &antibotv1.MetronomeRules{Clock: &antibotv1.ClockRules{MinCoherence: proto.Float64(0.5)}},
	}}.Restore(guard.Parts(), rules.Seeded())
	require.NoError(t, err)
	assert.True(t, same.Equal(sparse), "a set that holds less judges alike")
}

func TestASetKeepsNothingOfTheSnapshotItWasRestoredFrom(t *testing.T) {
	kept := rules.Named("rush", rules.AsIs)
	set, err := kept.Restore(guard.Parts(), rules.Seeded())
	require.NoError(t, err)

	kept.Rules.Cohort.CertainMembers = proto.Uint32(50)

	assert.Equal(t, uint32(6), set.TakeSnapshot().Rules.GetCohort().GetCertainMembers())
}
