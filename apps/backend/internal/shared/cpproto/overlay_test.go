package cpproto_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpproto"
)

func base() *planetv1.Rules {
	return &planetv1.Rules{
		Clicks: &planetv1.ClickRules{Tempo: proto.Float64(1), Burst: proto.Uint32(60)},
		Toll:   &planetv1.TollRules{Steps: []*planetv1.TollStep{{Share: 0.1, Slowdown: 1.5}}},
		Bonuses: &planetv1.BonusRules{
			Boxes: &planetv1.BoxPace{MinInterval: durationpb.New(4 * time.Minute), MaxInterval: durationpb.New(8 * time.Minute)},
			Quiz:  &planetv1.QuizRules{Enabled: proto.Bool(true)},
		},
		Frozen: proto.Bool(false),
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
		Bonuses: &planetv1.BonusRules{Quiz: &planetv1.QuizRules{Enabled: proto.Bool(false)}},
	})

	assert.True(t, overlaid.GetFrozen())
	assert.False(t, overlaid.GetBonuses().GetQuiz().GetEnabled())
	assert.Equal(t, 4*time.Minute, overlaid.GetBonuses().GetBoxes().GetMinInterval().AsDuration())
}

func TestOverlayWritesADurationWhole(t *testing.T) {
	overlaid := cpproto.Overlay(base(), &planetv1.Rules{
		Bonuses: &planetv1.BonusRules{Boxes: &planetv1.BoxPace{MinInterval: durationpb.New(90 * time.Second)}},
	})

	assert.Equal(t, 90*time.Second, overlaid.GetBonuses().GetBoxes().GetMinInterval().AsDuration())
	assert.Equal(t, 8*time.Minute, overlaid.GetBonuses().GetBoxes().GetMaxInterval().AsDuration())
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
