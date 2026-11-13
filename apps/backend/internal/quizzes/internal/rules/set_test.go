package rules_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

func rush() cprules.Snapshot[*quizzesv1.Rules] {
	return cprules.Snapshot[*quizzesv1.Rules]{Name: "rush", Rules: &quizzesv1.Rules{
		MinInterval: durationpb.New(time.Minute), MaxInterval: durationpb.New(2 * time.Minute),
		MaxRightAnswersPerHour: proto.Uint32(30),
	}}
}

func restored(t *testing.T, snapshot cprules.Snapshot[*quizzesv1.Rules]) cprules.Set[*quizzesv1.Rules] {
	t.Helper()

	set, err := snapshot.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	return set
}

func TestAZeroASetHoldsIsPlayed(t *testing.T) {
	quiet := cprules.Snapshot[*quizzesv1.Rules]{Name: "quiet", Rules: &quizzesv1.Rules{Enabled: proto.Bool(false)}}

	assert.False(t, restored(t, quiet).TakePlayedSnapshot().Rules.GetEnabled())
}

func TestADefaultThatCannotPlayIsRefusedWithTheValueNamed(t *testing.T) {
	d := durationpb.New
	broken := map[string]func(*quizzesv1.Rules){
		"minInterval":            func(r *quizzesv1.Rules) { r.MinInterval = d(0) },
		"maxInterval":            func(r *quizzesv1.Rules) { r.MaxInterval = d(time.Minute) },
		"maxRightAnswersPerHour": func(r *quizzesv1.Rules) { r.MaxRightAnswersPerHour = proto.Uint32(0) },
		"missRetry":              func(r *quizzesv1.Rules) { r.MissRetry = d(0) },
		"offerTtl":               func(r *quizzesv1.Rules) { r.OfferTtl = nil },
		"answerWindow":           func(r *quizzesv1.Rules) { r.AnswerWindow = d(0) },
		"leaderBias is -1":       func(r *quizzesv1.Rules) { r.LeaderBias = proto.Float64(-1) },
		"leaderBias is NaN":      func(r *quizzesv1.Rules) { r.LeaderBias = proto.Float64(math.NaN()) },
		"target is 0.2, and a chance of a right answer is between 0.34 and 0.99": func(r *quizzesv1.Rules) {
			r.Target = proto.Float64(0.2)
		},
		"target is 1":   func(r *quizzesv1.Rules) { r.Target = proto.Float64(1) },
		"width is -1":   func(r *quizzesv1.Rules) { r.Width = proto.Float64(-1) },
		"width is +Inf": func(r *quizzesv1.Rules) { r.Width = proto.Float64(math.Inf(1)) },
		"floor is 1.5, and a share is between 0 and 1": func(r *quizzesv1.Rules) { r.Floor = proto.Float64(1.5) },
		"floor is 0.8, over target (0.7)":              func(r *quizzesv1.Rules) { r.Floor = proto.Float64(0.8) },
		"floorAfter is 0, and must be above 0":         func(r *quizzesv1.Rules) { r.FloorAfter = proto.Uint32(0) },
		"explore is -0.1":                              func(r *quizzesv1.Rules) { r.Explore = proto.Float64(-0.1) },
		"explore is 2":                                 func(r *quizzesv1.Rules) { r.Explore = proto.Float64(2) },
		"activeWithin":                                 func(r *quizzesv1.Rules) { r.ActiveWithin = d(0) },
		"welcome.within":                               func(r *quizzesv1.Rules) { r.Welcome.Within = d(-time.Minute) },
	}
	for reason, breaks := range broken {
		t.Run(reason, func(t *testing.T) {
			_, err := rules.Named(cprules.Default, breaks).Restore(parts.All(), rules.Seeded())

			require.ErrorIs(t, err, cprules.ErrInvalidSet)
			assert.Contains(t, err.Error(), reason)
		})
	}
}

func TestASetIsCheckedOverDefault(t *testing.T) {
	slower := cprules.Snapshot[*quizzesv1.Rules]{Name: "slower", Rules: &quizzesv1.Rules{
		MinInterval: durationpb.New(10 * time.Minute),
	}}

	_, err := slower.Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "maxInterval is 8m0s, under minInterval (10m0s)")
}

func TestAFloorIsCheckedAgainstDefaultsTarget(t *testing.T) {
	harsh := cprules.Snapshot[*quizzesv1.Rules]{Name: "harsh", Rules: &quizzesv1.Rules{Floor: proto.Float64(0.75)}}

	_, err := harsh.Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "floor is 0.75, over target (0.7)")
}

func TestDefaultPlaysOnlyWhatItHolds(t *testing.T) {
	_, err := rules.Named(cprules.Default, func(r *quizzesv1.Rules) { r.MinInterval = nil }).Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "minInterval is 0s")
}

func TestEveryReasonIsSaid(t *testing.T) {
	_, err := rules.Named(cprules.Default, func(r *quizzesv1.Rules) {
		r.MissRetry = nil
		r.AnswerWindow = nil
	}).Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "missRetry is 0s")
	assert.Contains(t, err.Error(), "answerWindow is 0s")
}

func TestEveryValueThatCanBeZeroIsADefault(t *testing.T) {
	set := rules.Named(cprules.Default, func(r *quizzesv1.Rules) {
		r.Enabled = proto.Bool(false)
		r.LeaderBias = proto.Float64(0)
		r.Width = proto.Float64(0)
		r.Floor = proto.Float64(0)
		r.Explore = proto.Float64(0)
		r.MaxInterval = r.MinInterval
		r.Welcome = nil
		r.Frozen = nil
	})

	_, err := set.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
}
