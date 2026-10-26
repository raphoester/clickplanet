//go:build testing

package rules

import (
	"time"

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
				Enforce:        true,
				BanDurations:   []*durationpb.Duration{d(24 * time.Hour), d(7 * 24 * time.Hour), d(3 * 365 * 24 * time.Hour)},
				ReflagInterval: d(5 * time.Minute),
			},
			Jury: &antibotv1.JuryRules{MinSuspects: 2, SuspicionWindow: d(10 * time.Minute)},
			Retaker: &antibotv1.RetakerRules{
				Enabled:        true,
				ReactionWindow: d(5 * time.Second),
				MinReactions:   20,
				MaxSpread:      d(300 * time.Millisecond),
				MaxMedian:      d(250 * time.Millisecond),
				MinTiles:       15,
				RoamMedian:     d(600 * time.Millisecond),
			},
			Sequencer: &antibotv1.SequencerRules{Enabled: true, MinSteps: 40, MinShare: 0.75, CertainSteps: 200, CertainShare: 0.95},
			Metronome: &antibotv1.MetronomeRules{
				Enabled:       true,
				MaxGap:        d(7 * time.Second),
				MaxSpread:     d(400 * time.Millisecond),
				MinClicks:     120,
				CertainFor:    d(30 * time.Minute),
				CertainClicks: 900,
				Shape:         &antibotv1.ShapeRules{MaxGap: d(10 * time.Second), Clicks: 500, CertainClicks: 1000},
				Clock: &antibotv1.ClockRules{
					Period: d(time.Second), Clicks: 120, CertainClicks: 600, CertainFor: d(30 * time.Minute),
				},
				Stamina: &antibotv1.StaminaRules{
					Clicks: 40, Window: d(6 * time.Hour), MinBusy: d(5*time.Hour + 30*time.Minute), CertainBusy: d(0),
				},
			},
			Defender: &antibotv1.DefenderRules{
				Enabled: true, RetakeWindow: d(2 * time.Minute), MinClicks: 40, MinShare: 0.6, CertainClicks: 200,
			},
			Catcher: &antibotv1.CatcherRules{
				Enabled:       true,
				MinCatches:    5,
				MaxMedian:     d(3 * time.Second),
				CertainMedian: d(1500 * time.Millisecond),
				Foreign:       &antibotv1.ForeignRules{Window: d(time.Hour)},
			},
			Cohort: &antibotv1.CohortRules{
				Enabled:        true,
				StartWindow:    d(5 * time.Second),
				MinClicks:      20,
				MinFlagShare:   0.9,
				RateRatio:      1.5,
				LengthRatio:    1.2,
				QuietAfter:     d(time.Minute),
				MinMembers:     2,
				CertainCohorts: 3,
				CertainMembers: 6,
				ChainWindow:    d(30 * time.Minute),
			},
			Scraper: &antibotv1.ScraperRules{Enabled: true, MinMaps: 5, CertainMaps: 15, CertainOffMap: 2},
			Churner: &antibotv1.ChurnerRules{
				Enabled: true,
				Window:  d(time.Hour),
				V6:      &antibotv1.AccountBounds{MinAccounts: 4, CertainAccounts: 6},
				V4:      &antibotv1.AccountBounds{MinAccounts: 10, CertainAccounts: 20},
				Relay: &antibotv1.RelayRules{
					Handoff:      d(90 * time.Second),
					MaxLife:      d(5 * time.Minute),
					MinClicks:    20,
					MinFlagShare: 0.9,
					MinLinks:     3,
					CertainLinks: 6,
				},
			},
			Hopper: &antibotv1.HopperRules{
				Enabled:      true,
				MinAngle:     45,
				MinGap:       d(100 * time.Millisecond),
				MaxGap:       d(30 * time.Second),
				MinSteps:     40,
				MinShare:     0.4,
				CertainSteps: 200,
			},
			Rotator: &antibotv1.RotatorRules{
				Enabled:     true,
				Window:      d(24 * time.Hour),
				MinSeconds:  15,
				MinByChance: 90,
				MaxShare:    0.05,
				MinSlots:    24,
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
