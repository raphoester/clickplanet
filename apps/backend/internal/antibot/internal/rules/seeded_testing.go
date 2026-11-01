//go:build testing

package rules

import (
	"time"

	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/types/known/durationpb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
)

// Seeded is the set the migration keeps as default: production's rules on 26 October 2026.
func Seeded() Snapshot {
	d := durationpb.New
	return Snapshot{
		Name: Default,
		Rules: &antibotv1.Rules{
			Bans: &antibotv1.BanRules{
				Enforce:        proto.Bool(true),
				BanDurations:   []*durationpb.Duration{d(24 * time.Hour), d(7 * 24 * time.Hour), d(3 * 365 * 24 * time.Hour)},
				ReflagInterval: d(5 * time.Minute),
			},
			Jury: &antibotv1.JuryRules{MinSuspects: proto.Uint32(2), SuspicionWindow: d(10 * time.Minute)},
			Retaker: &antibotv1.RetakerRules{
				Enabled:        proto.Bool(true),
				ReactionWindow: d(5 * time.Second),
				MinReactions:   proto.Uint32(20),
				MaxSpread:      d(300 * time.Millisecond),
				MaxMedian:      d(250 * time.Millisecond),
				MinTiles:       proto.Uint32(15),
				RoamMedian:     d(600 * time.Millisecond),
				CertainTiles:   proto.Uint32(0),
			},
			Sequencer: &antibotv1.SequencerRules{Enabled: proto.Bool(true), MinSteps: proto.Uint32(40), MinShare: proto.Float64(0.75), CertainSteps: proto.Uint32(200), CertainShare: proto.Float64(0.95)},
			Metronome: &antibotv1.MetronomeRules{
				Enabled:       proto.Bool(true),
				MaxGap:        d(7 * time.Second),
				MaxSpread:     d(400 * time.Millisecond),
				MinClicks:     proto.Uint32(120),
				CertainFor:    d(30 * time.Minute),
				CertainClicks: proto.Uint32(900),
				Shape:         &antibotv1.ShapeRules{MaxGap: d(10 * time.Second), Clicks: proto.Uint32(500), CertainClicks: proto.Uint32(1000)},
				Clock: &antibotv1.ClockRules{
					Period: d(time.Second), Clicks: proto.Uint32(120), CertainClicks: proto.Uint32(600), CertainFor: d(30 * time.Minute),
				},
				Stamina: &antibotv1.StaminaRules{
					Clicks: proto.Uint32(40), Window: d(6 * time.Hour), MinBusy: d(5*time.Hour + 30*time.Minute), CertainBusy: d(0),
				},
			},
			Defender: &antibotv1.DefenderRules{
				Enabled: proto.Bool(true), RetakeWindow: d(2 * time.Minute), MinClicks: proto.Uint32(40), MinShare: proto.Float64(0.6), CertainClicks: proto.Uint32(200),
				CertainShare: proto.Float64(0),
			},
			Catcher: &antibotv1.CatcherRules{
				Enabled:       proto.Bool(true),
				MinCatches:    proto.Uint32(5),
				MaxMedian:     d(3 * time.Second),
				CertainMedian: d(1500 * time.Millisecond),
				Foreign:       &antibotv1.ForeignRules{Window: d(time.Hour), MinClaims: proto.Uint32(0), CertainClaims: proto.Uint32(0)},
			},
			Cohort: &antibotv1.CohortRules{
				Enabled:        proto.Bool(true),
				StartWindow:    d(5 * time.Second),
				MinClicks:      proto.Uint32(20),
				MinFlagShare:   proto.Float64(0.9),
				RateRatio:      proto.Float64(1.5),
				LengthRatio:    proto.Float64(1.2),
				QuietAfter:     d(time.Minute),
				MinMembers:     proto.Uint32(2),
				CertainCohorts: proto.Uint32(3),
				CertainMembers: proto.Uint32(6),
				ChainWindow:    d(30 * time.Minute),
			},
			Scraper: &antibotv1.ScraperRules{Enabled: proto.Bool(true), MinMaps: proto.Float64(5), CertainMaps: proto.Float64(15), CertainOffMap: proto.Uint32(2)},
			Churner: &antibotv1.ChurnerRules{
				Enabled: proto.Bool(true),
				Window:  d(time.Hour),
				V6:      &antibotv1.AccountBounds{MinAccounts: proto.Uint32(4), CertainAccounts: proto.Uint32(6)},
				V4:      &antibotv1.AccountBounds{MinAccounts: proto.Uint32(10), CertainAccounts: proto.Uint32(20)},
				Relay: &antibotv1.RelayRules{
					Handoff:      d(90 * time.Second),
					MaxLife:      d(5 * time.Minute),
					MinClicks:    proto.Uint32(20),
					MinFlagShare: proto.Float64(0.9),
					MinLinks:     proto.Uint32(3),
					CertainLinks: proto.Uint32(6),
				},
			},
			Hopper: &antibotv1.HopperRules{
				Enabled:      proto.Bool(true),
				MinAngle:     proto.Float64(45),
				MinGap:       d(100 * time.Millisecond),
				MaxGap:       d(30 * time.Second),
				MinSteps:     proto.Uint32(40),
				MinShare:     proto.Float64(0.4),
				CertainSteps: proto.Uint32(200),
				CertainShare: proto.Float64(0),
			},
			Rotator: &antibotv1.RotatorRules{
				Enabled:      proto.Bool(true),
				Window:       d(24 * time.Hour),
				MinSeconds:   proto.Uint32(15),
				MinByChance:  proto.Float64(90),
				MaxShare:     proto.Float64(0.05),
				MinSlots:     proto.Uint32(24),
				CertainSlots: proto.Uint32(0),
			},
		},
	}
}

func Named(name SetName, change func(*antibotv1.Rules)) Snapshot {
	kept := Seeded()
	kept.Name = name
	change(kept.Rules)
	return kept
}

func AsIs(*antibotv1.Rules) {}
