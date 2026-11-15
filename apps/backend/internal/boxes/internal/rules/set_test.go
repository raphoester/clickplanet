package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

func rush() cprules.Snapshot[*boxesv1.Rules] {
	return cprules.Snapshot[*boxesv1.Rules]{Name: "rush", Rules: &boxesv1.Rules{
		MinInterval: durationpb.New(time.Minute), MaxInterval: durationpb.New(2 * time.Minute),
		MaxBoxesPerHour: proto.Uint32(30),
	}}
}

func restored(t *testing.T, snapshot cprules.Snapshot[*boxesv1.Rules]) cprules.Set[*boxesv1.Rules] {
	t.Helper()

	set, err := snapshot.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	return set
}

func TestAZeroASetHoldsIsPlayed(t *testing.T) {
	thawed := cprules.Snapshot[*boxesv1.Rules]{Name: "thawed", Rules: &boxesv1.Rules{Frozen: proto.Bool(false)}}

	assert.False(t, restored(t, thawed).TakePlayedSnapshot().Rules.GetFrozen())
}

func TestADefaultThatCannotPlayIsRefusedWithTheValueNamed(t *testing.T) {
	d := durationpb.New
	broken := map[string]func(*boxesv1.Rules){
		"minInterval is 0s":     func(r *boxesv1.Rules) { r.MinInterval = d(0) },
		"maxInterval is 1m0s":   func(r *boxesv1.Rules) { r.MaxInterval = d(time.Minute) },
		"maxBoxesPerHour is 0":  func(r *boxesv1.Rules) { r.MaxBoxesPerHour = proto.Uint32(0) },
		"offerTtl is 0s":        func(r *boxesv1.Rules) { r.OfferTtl = nil },
		"missRetry is 0s":       func(r *boxesv1.Rules) { r.MissRetry = d(0) },
		"activeWithin is 0s":    func(r *boxesv1.Rules) { r.ActiveWithin = d(0) },
		"welcome.within is -1m": func(r *boxesv1.Rules) { r.Welcome.Within = d(-time.Minute) },
		"welcome.boxAfter is -": func(r *boxesv1.Rules) { r.Welcome.BoxAfter = d(-time.Minute) },
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
	slower := cprules.Snapshot[*boxesv1.Rules]{Name: "slower", Rules: &boxesv1.Rules{
		MinInterval: durationpb.New(10 * time.Minute),
	}}

	_, err := slower.Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "maxInterval is 8m0s, under minInterval (10m0s)")
}

func TestDefaultPlaysOnlyWhatItHolds(t *testing.T) {
	_, err := rules.Named(cprules.Default, func(r *boxesv1.Rules) { r.MinInterval = nil }).
		Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "minInterval is 0s")
}

func TestEveryReasonIsSaid(t *testing.T) {
	_, err := rules.Named(cprules.Default, func(r *boxesv1.Rules) {
		r.OfferTtl = nil
		r.MissRetry = nil
	}).Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "offerTtl is 0s")
	assert.Contains(t, err.Error(), "missRetry is 0s")
}

func TestEveryValueThatCanBeZeroIsADefault(t *testing.T) {
	set := rules.Named(cprules.Default, func(r *boxesv1.Rules) {
		r.Welcome = nil
		r.MaxInterval = r.GetMinInterval()
		r.Frozen = nil
	})

	_, err := set.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
}
