//go:build testing

package rules

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/catcher"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/churner"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/cohort"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/defender"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/hopper"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/jury"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/metronome"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/retaker"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rotator"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/scraper"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/sequencer"
)

// Seeded is the set the migration keeps as default: production's rules on 26 October 2026.
func Seeded() Snapshot {
	return Snapshot{
		Name: Default,
		Bans: bans.Rules{
			Enforce:        true,
			BanDurations:   []time.Duration{24 * time.Hour, 7 * 24 * time.Hour, 3 * 365 * 24 * time.Hour},
			ReflagInterval: 5 * time.Minute,
		},
		Jury: jury.Rules{MinSuspects: 2, SuspicionWindow: 10 * time.Minute},
		Retaker: retaker.Rules{
			ReactionWindow: 5 * time.Second,
			MinReactions:   20,
			MaxSpread:      300 * time.Millisecond,
			MaxMedian:      250 * time.Millisecond,
			MinTiles:       15,
			RoamMedian:     600 * time.Millisecond,
		},
		Sequencer: sequencer.Rules{MinSteps: 40, MinShare: 0.75, CertainSteps: 200, CertainShare: 0.95},
		Metronome: metronome.Rules{
			MaxGap:        7 * time.Second,
			MaxSpread:     400 * time.Millisecond,
			MinClicks:     120,
			CertainFor:    30 * time.Minute,
			CertainClicks: 900,
			Shape:         metronome.ShapeRules{MaxGap: 10 * time.Second, Clicks: 500, CertainClicks: 1000},
			Clock: metronome.ClockRules{
				Period: time.Second, Clicks: 120, CertainClicks: 600, CertainFor: 30 * time.Minute,
			},
			Stamina: metronome.StaminaRules{Clicks: 40, Window: 6 * time.Hour, MinBusy: 5*time.Hour + 30*time.Minute},
		},
		Defender: defender.Rules{RetakeWindow: 2 * time.Minute, MinClicks: 40, MinShare: 0.6, CertainClicks: 200},
		Catcher: catcher.Rules{
			MinCatches:    5,
			MaxMedian:     3 * time.Second,
			CertainMedian: 1500 * time.Millisecond,
			Foreign:       catcher.ForeignRules{Window: time.Hour},
		},
		Cohort: cohort.Rules{
			StartWindow:    5 * time.Second,
			MinClicks:      20,
			MinFlagShare:   0.9,
			RateRatio:      1.5,
			LengthRatio:    1.2,
			QuietAfter:     time.Minute,
			MinMembers:     2,
			CertainCohorts: 3,
			CertainMembers: 6,
			ChainWindow:    30 * time.Minute,
		},
		Scraper: scraper.Rules{MinMaps: 5, CertainMaps: 15, CertainOffMap: 2},
		Churner: churner.Rules{
			Window: time.Hour,
			V6:     churner.Bounds{MinAccounts: 4, CertainAccounts: 6},
			V4:     churner.Bounds{MinAccounts: 10, CertainAccounts: 20},
			Relay: churner.RelayRules{
				Handoff:      90 * time.Second,
				MaxLife:      5 * time.Minute,
				MinClicks:    20,
				MinFlagShare: 0.9,
				MinLinks:     3,
				CertainLinks: 6,
			},
		},
		Hopper: hopper.Rules{
			MinAngle:     45,
			MinGap:       100 * time.Millisecond,
			MaxGap:       30 * time.Second,
			MinSteps:     40,
			MinShare:     0.4,
			CertainSteps: 200,
		},
		Rotator: rotator.Rules{
			Window:      24 * time.Hour,
			MinSeconds:  15,
			MinByChance: 90,
			MaxShare:    0.05,
			MinSlots:    24,
		},
	}
}

// Named is the seeded set under another name, changed as asked.
func Named(name SetName, change func(*Snapshot)) Snapshot {
	kept := Seeded()
	kept.Name = name
	change(&kept)
	return kept
}

func AsIs(*Snapshot) {}
