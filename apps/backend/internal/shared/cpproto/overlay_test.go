package cpproto_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpproto"
)

func base() *planetv1.Rules {
	return &planetv1.Rules{
		Clicks: &planetv1.ClickRules{
			Tempo: proto.Float64(1), Burst: proto.Uint32(60), FastStart: durationpb.New(30 * time.Minute),
		},
		Toll:    &planetv1.TollRules{Steps: []*planetv1.TollStep{{Share: 0.1, Slowdown: 1.5}}},
		Charges: &planetv1.ChargeRules{Bomb: &planetv1.BombRules{Rings: proto.Float64(4), Off: proto.Bool(true)}},
		Frozen:  proto.Bool(false),
	}
}

func TestOverlayKeepsWhatTopLeavesUnset(t *testing.T) {
	overlaid := cpproto.Overlay(base(), &planetv1.Rules{Clicks: &planetv1.ClickRules{Tempo: proto.Float64(3)}})

	want := base()
	want.Clicks.Tempo = proto.Float64(3)
	assert.True(t, proto.Equal(want, overlaid), "%v", overlaid)
}

func TestOverlayWritesAZeroTopHolds(t *testing.T) {
	overlaid := cpproto.Overlay(base(), &planetv1.Rules{
		Frozen:  proto.Bool(true),
		Charges: &planetv1.ChargeRules{Bomb: &planetv1.BombRules{Off: proto.Bool(false)}},
	})

	assert.True(t, overlaid.GetFrozen())
	assert.False(t, overlaid.GetCharges().GetBomb().GetOff())
	assert.InDelta(t, 4, overlaid.GetCharges().GetBomb().GetRings(), 0)
}

func TestOverlayWritesADurationWhole(t *testing.T) {
	overlaid := cpproto.Overlay(base(), &planetv1.Rules{
		Clicks: &planetv1.ClickRules{FastStart: durationpb.New(90 * time.Second)},
	})

	assert.Equal(t, 90*time.Second, overlaid.GetClicks().GetFastStart().AsDuration())
	assert.Equal(t, uint32(60), overlaid.GetClicks().GetBurst())
}

func TestAMessageOfListsOnlyIsOneValueEvenEmpty(t *testing.T) {
	assert.Empty(t, cpproto.Overlay(base(), &planetv1.Rules{Toll: &planetv1.TollRules{}}).GetToll().GetSteps())

	toll := &planetv1.TollRules{Steps: []*planetv1.TollStep{{Share: 0.5, Slowdown: 2}}}
	overlaid := cpproto.Overlay(base(), &planetv1.Rules{Toll: toll})
	assert.True(t, proto.Equal(toll, overlaid.GetToll()), "%v", overlaid.GetToll())
}

func TestAListBesideOtherValuesReplacesOnlyWhenItHoldsAnItem(t *testing.T) {
	kept := &antibotv1.Rules{Bans: &antibotv1.BanRules{
		Enforce:      proto.Bool(true),
		BanDurations: []*durationpb.Duration{durationpb.New(time.Hour), durationpb.New(24 * time.Hour)},
	}}

	enforce := cpproto.Overlay(kept, &antibotv1.Rules{Bans: &antibotv1.BanRules{Enforce: proto.Bool(false)}})
	assert.False(t, enforce.GetBans().GetEnforce())
	assert.Len(t, enforce.GetBans().GetBanDurations(), 2)

	ladder := cpproto.Overlay(kept, &antibotv1.Rules{Bans: &antibotv1.BanRules{
		BanDurations: []*durationpb.Duration{durationpb.New(time.Minute)},
	}})
	assert.True(t, ladder.GetBans().GetEnforce())
	assert.Len(t, ladder.GetBans().GetBanDurations(), 1)
	assert.Equal(t, time.Minute, ladder.GetBans().GetBanDurations()[0].AsDuration())
}

func TestOverlayChangesNeitherBaseNorTop(t *testing.T) {
	kept, top := base(), &planetv1.Rules{Clicks: &planetv1.ClickRules{Tempo: proto.Float64(3)}}

	overlaid := cpproto.Overlay(kept, top)
	overlaid.Clicks.Burst = proto.Uint32(1)
	overlaid.Clicks.Tempo = proto.Float64(9)

	assert.True(t, proto.Equal(base(), kept), "%v", kept)
	assert.InDelta(t, 3.0, top.GetClicks().GetTempo(), 1e-9)
}

func TestOverlayMergesTheOneofCaseBothHold(t *testing.T) {
	base := &planetv1.Seed{Pattern: &planetv1.Seed_Capitals{Capitals: &planetv1.CapitalSeed{Shields: proto.Uint32(3)}}}

	overlaid := cpproto.Overlay(base, &planetv1.Seed{Pattern: &planetv1.Seed_Capitals{Capitals: &planetv1.CapitalSeed{}}})

	assert.True(t, proto.Equal(base, overlaid), "%v", overlaid)
}

func TestOverlayPlaysTopsOneofCaseAloneOverAnother(t *testing.T) {
	ends := timestamppb.New(time.Date(2026, 11, 12, 22, 0, 0, 0, time.UTC))
	base := &scoringv1.CloseRoundResponse{Outcome: &scoringv1.CloseRoundResponse_ScoredAt{ScoredAt: ends}}
	top := &scoringv1.CloseRoundResponse{Outcome: &scoringv1.CloseRoundResponse_MissedAt{MissedAt: ends}}

	overlaid := cpproto.Overlay(base, top)

	assert.True(t, proto.Equal(top, overlaid), "nothing of base's other case: %v", overlaid)
}
