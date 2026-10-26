package rulesmessage

import (
	"math"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
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
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/scraper"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/sequencer"
)

// Document writes every value, its zeros too, so a set kept reads whole.
var Document = protojson.MarshalOptions{EmitUnpopulated: true}

func Of(set rules.Snapshot) *antibotv1.Rules {
	return &antibotv1.Rules{
		Bans: &antibotv1.BanRules{
			Enforce:        set.Bans.Enforce,
			BanDurations:   durations(set.Bans.BanDurations),
			ReflagInterval: durationpb.New(set.Bans.ReflagInterval),
		},
		Jury: &antibotv1.JuryRules{
			MinSuspects:     count(set.Jury.MinSuspects),
			SuspicionWindow: durationpb.New(set.Jury.SuspicionWindow),
		},
		Retaker:   retakerOf(set.Retaker),
		Sequencer: sequencerOf(set.Sequencer),
		Metronome: metronomeOf(set.Metronome),
		Defender:  defenderOf(set.Defender),
		Catcher:   catcherOf(set.Catcher),
		Cohort:    cohortOf(set.Cohort),
		Scraper: &antibotv1.ScraperRules{
			MinMaps:       set.Scraper.MinMaps,
			CertainMaps:   set.Scraper.CertainMaps,
			CertainOffMap: count(set.Scraper.CertainOffMap),
		},
		Churner: churnerOf(set.Churner),
		Hopper:  hopperOf(set.Hopper),
		Rotator: rotatorOf(set.Rotator),
	}
}

func retakerOf(r retaker.Rules) *antibotv1.RetakerRules {
	return &antibotv1.RetakerRules{
		ReactionWindow: durationpb.New(r.ReactionWindow),
		MinReactions:   count(r.MinReactions),
		MaxSpread:      durationpb.New(r.MaxSpread),
		MaxMedian:      durationpb.New(r.MaxMedian),
		MinTiles:       count(r.MinTiles),
		RoamMedian:     durationpb.New(r.RoamMedian),
		CertainTiles:   count(r.CertainTiles),
	}
}

func sequencerOf(r sequencer.Rules) *antibotv1.SequencerRules {
	return &antibotv1.SequencerRules{
		MinSteps:     count(r.MinSteps),
		MinShare:     r.MinShare,
		CertainSteps: count(r.CertainSteps),
		CertainShare: r.CertainShare,
	}
}

func metronomeOf(r metronome.Rules) *antibotv1.MetronomeRules {
	return &antibotv1.MetronomeRules{
		MaxGap:        durationpb.New(r.MaxGap),
		MaxSpread:     durationpb.New(r.MaxSpread),
		MinClicks:     count(r.MinClicks),
		CertainFor:    durationpb.New(r.CertainFor),
		CertainClicks: count(r.CertainClicks),
		Shape: &antibotv1.ShapeRules{
			MaxGap:        durationpb.New(r.Shape.MaxGap),
			Clicks:        count(r.Shape.Clicks),
			MaxSkew:       copied(r.Shape.MaxSkew),
			CertainClicks: count(r.Shape.CertainClicks),
			CertainSkew:   copied(r.Shape.CertainSkew),
		},
		Clock: &antibotv1.ClockRules{
			Period:           durationpb.New(r.Clock.Period),
			Clicks:           count(r.Clock.Clicks),
			MinCoherence:     copied(r.Clock.MinCoherence),
			CertainClicks:    count(r.Clock.CertainClicks),
			CertainFor:       durationpb.New(r.Clock.CertainFor),
			CertainCoherence: copied(r.Clock.CertainCoherence),
		},
		Stamina: &antibotv1.StaminaRules{
			Clicks:      count(r.Stamina.Clicks),
			Window:      durationpb.New(r.Stamina.Window),
			MinBusy:     durationpb.New(r.Stamina.MinBusy),
			CertainBusy: durationpb.New(r.Stamina.CertainBusy),
		},
	}
}

func defenderOf(r defender.Rules) *antibotv1.DefenderRules {
	return &antibotv1.DefenderRules{
		RetakeWindow:  durationpb.New(r.RetakeWindow),
		MinClicks:     count(r.MinClicks),
		MinShare:      r.MinShare,
		CertainClicks: count(r.CertainClicks),
		CertainShare:  r.CertainShare,
	}
}

func catcherOf(r catcher.Rules) *antibotv1.CatcherRules {
	return &antibotv1.CatcherRules{
		MinCatches:    count(r.MinCatches),
		MaxMedian:     durationpb.New(r.MaxMedian),
		CertainMedian: durationpb.New(r.CertainMedian),
		Foreign: &antibotv1.ForeignRules{
			Window:        durationpb.New(r.Foreign.Window),
			MinClaims:     count(r.Foreign.MinClaims),
			CertainClaims: count(r.Foreign.CertainClaims),
		},
	}
}

func cohortOf(r cohort.Rules) *antibotv1.CohortRules {
	return &antibotv1.CohortRules{
		StartWindow:    durationpb.New(r.StartWindow),
		MinClicks:      count(r.MinClicks),
		MinFlagShare:   r.MinFlagShare,
		RateRatio:      r.RateRatio,
		LengthRatio:    r.LengthRatio,
		QuietAfter:     durationpb.New(r.QuietAfter),
		MinMembers:     count(r.MinMembers),
		CertainCohorts: count(r.CertainCohorts),
		CertainMembers: count(r.CertainMembers),
		ChainWindow:    durationpb.New(r.ChainWindow),
	}
}

func churnerOf(r churner.Rules) *antibotv1.ChurnerRules {
	return &antibotv1.ChurnerRules{
		Window: durationpb.New(r.Window),
		V6:     &antibotv1.AccountBounds{MinAccounts: count(r.V6.MinAccounts), CertainAccounts: count(r.V6.CertainAccounts)},
		V4:     &antibotv1.AccountBounds{MinAccounts: count(r.V4.MinAccounts), CertainAccounts: count(r.V4.CertainAccounts)},
		Relay: &antibotv1.RelayRules{
			Handoff:      durationpb.New(r.Relay.Handoff),
			MaxLife:      durationpb.New(r.Relay.MaxLife),
			MinClicks:    count(r.Relay.MinClicks),
			MinFlagShare: r.Relay.MinFlagShare,
			MinLinks:     count(r.Relay.MinLinks),
			CertainLinks: count(r.Relay.CertainLinks),
		},
	}
}

func hopperOf(r hopper.Rules) *antibotv1.HopperRules {
	return &antibotv1.HopperRules{
		MinAngle:     r.MinAngle,
		MinGap:       durationpb.New(r.MinGap),
		MaxGap:       durationpb.New(r.MaxGap),
		MinSteps:     count(r.MinSteps),
		MinShare:     r.MinShare,
		CertainSteps: count(r.CertainSteps),
		CertainShare: r.CertainShare,
	}
}

func rotatorOf(r rotator.Rules) *antibotv1.RotatorRules {
	return &antibotv1.RotatorRules{
		Window:       durationpb.New(r.Window),
		MinSeconds:   count(r.MinSeconds),
		MinByChance:  r.MinByChance,
		MaxShare:     r.MaxShare,
		MinSlots:     count(r.MinSlots),
		CertainSlots: count(r.CertainSlots),
	}
}

// SnapshotOf reads a value the message leaves out as zero, which Restore refuses where zero cannot be judged with.
func SnapshotOf(name rules.SetName, m *antibotv1.Rules) rules.Snapshot {
	return rules.Snapshot{
		Name: name,
		Bans: bans.Rules{
			Enforce:        m.GetBans().GetEnforce(),
			BanDurations:   durationsOf(m.GetBans().GetBanDurations()),
			ReflagInterval: m.GetBans().GetReflagInterval().AsDuration(),
		},
		Jury: jury.Rules{
			MinSuspects:     int(m.GetJury().GetMinSuspects()),
			SuspicionWindow: m.GetJury().GetSuspicionWindow().AsDuration(),
		},
		Retaker:   retakerRules(m.GetRetaker()),
		Sequencer: sequencerRules(m.GetSequencer()),
		Metronome: metronomeRules(m.GetMetronome()),
		Defender:  defenderRules(m.GetDefender()),
		Catcher:   catcherRules(m.GetCatcher()),
		Cohort:    cohortRules(m.GetCohort()),
		Scraper: scraper.Rules{
			MinMaps:       m.GetScraper().GetMinMaps(),
			CertainMaps:   m.GetScraper().GetCertainMaps(),
			CertainOffMap: int(m.GetScraper().GetCertainOffMap()),
		},
		Churner: churnerRules(m.GetChurner()),
		Hopper:  hopperRules(m.GetHopper()),
		Rotator: rotatorRules(m.GetRotator()),
	}
}

func retakerRules(m *antibotv1.RetakerRules) retaker.Rules {
	return retaker.Rules{
		ReactionWindow: m.GetReactionWindow().AsDuration(),
		MinReactions:   int(m.GetMinReactions()),
		MaxSpread:      m.GetMaxSpread().AsDuration(),
		MaxMedian:      m.GetMaxMedian().AsDuration(),
		MinTiles:       int(m.GetMinTiles()),
		RoamMedian:     m.GetRoamMedian().AsDuration(),
		CertainTiles:   int(m.GetCertainTiles()),
	}
}

func sequencerRules(m *antibotv1.SequencerRules) sequencer.Rules {
	return sequencer.Rules{
		MinSteps:     int(m.GetMinSteps()),
		MinShare:     m.GetMinShare(),
		CertainSteps: int(m.GetCertainSteps()),
		CertainShare: m.GetCertainShare(),
	}
}

func metronomeRules(m *antibotv1.MetronomeRules) metronome.Rules {
	var maxSkew, certainSkew, minCoherence, certainCoherence *float64
	if shape := m.GetShape(); shape != nil {
		maxSkew, certainSkew = copied(shape.MaxSkew), copied(shape.CertainSkew)
	}
	if clock := m.GetClock(); clock != nil {
		minCoherence, certainCoherence = copied(clock.MinCoherence), copied(clock.CertainCoherence)
	}

	return metronome.Rules{
		MaxGap:        m.GetMaxGap().AsDuration(),
		MaxSpread:     m.GetMaxSpread().AsDuration(),
		MinClicks:     int(m.GetMinClicks()),
		CertainFor:    m.GetCertainFor().AsDuration(),
		CertainClicks: int(m.GetCertainClicks()),
		Shape: metronome.ShapeRules{
			MaxGap:        m.GetShape().GetMaxGap().AsDuration(),
			Clicks:        int(m.GetShape().GetClicks()),
			MaxSkew:       maxSkew,
			CertainClicks: int(m.GetShape().GetCertainClicks()),
			CertainSkew:   certainSkew,
		},
		Clock: metronome.ClockRules{
			Period:           m.GetClock().GetPeriod().AsDuration(),
			Clicks:           int(m.GetClock().GetClicks()),
			MinCoherence:     minCoherence,
			CertainClicks:    int(m.GetClock().GetCertainClicks()),
			CertainFor:       m.GetClock().GetCertainFor().AsDuration(),
			CertainCoherence: certainCoherence,
		},
		Stamina: metronome.StaminaRules{
			Clicks:      int(m.GetStamina().GetClicks()),
			Window:      m.GetStamina().GetWindow().AsDuration(),
			MinBusy:     m.GetStamina().GetMinBusy().AsDuration(),
			CertainBusy: m.GetStamina().GetCertainBusy().AsDuration(),
		},
	}
}

func defenderRules(m *antibotv1.DefenderRules) defender.Rules {
	return defender.Rules{
		RetakeWindow:  m.GetRetakeWindow().AsDuration(),
		MinClicks:     int(m.GetMinClicks()),
		MinShare:      m.GetMinShare(),
		CertainClicks: int(m.GetCertainClicks()),
		CertainShare:  m.GetCertainShare(),
	}
}

func catcherRules(m *antibotv1.CatcherRules) catcher.Rules {
	return catcher.Rules{
		MinCatches:    int(m.GetMinCatches()),
		MaxMedian:     m.GetMaxMedian().AsDuration(),
		CertainMedian: m.GetCertainMedian().AsDuration(),
		Foreign: catcher.ForeignRules{
			Window:        m.GetForeign().GetWindow().AsDuration(),
			MinClaims:     int(m.GetForeign().GetMinClaims()),
			CertainClaims: int(m.GetForeign().GetCertainClaims()),
		},
	}
}

func cohortRules(m *antibotv1.CohortRules) cohort.Rules {
	return cohort.Rules{
		StartWindow:    m.GetStartWindow().AsDuration(),
		MinClicks:      int(m.GetMinClicks()),
		MinFlagShare:   m.GetMinFlagShare(),
		RateRatio:      m.GetRateRatio(),
		LengthRatio:    m.GetLengthRatio(),
		QuietAfter:     m.GetQuietAfter().AsDuration(),
		MinMembers:     int(m.GetMinMembers()),
		CertainCohorts: int(m.GetCertainCohorts()),
		CertainMembers: int(m.GetCertainMembers()),
		ChainWindow:    m.GetChainWindow().AsDuration(),
	}
}

func churnerRules(m *antibotv1.ChurnerRules) churner.Rules {
	return churner.Rules{
		Window: m.GetWindow().AsDuration(),
		V6: churner.Bounds{
			MinAccounts: int(m.GetV6().GetMinAccounts()), CertainAccounts: int(m.GetV6().GetCertainAccounts()),
		},
		V4: churner.Bounds{
			MinAccounts: int(m.GetV4().GetMinAccounts()), CertainAccounts: int(m.GetV4().GetCertainAccounts()),
		},
		Relay: churner.RelayRules{
			Handoff:      m.GetRelay().GetHandoff().AsDuration(),
			MaxLife:      m.GetRelay().GetMaxLife().AsDuration(),
			MinClicks:    int(m.GetRelay().GetMinClicks()),
			MinFlagShare: m.GetRelay().GetMinFlagShare(),
			MinLinks:     int(m.GetRelay().GetMinLinks()),
			CertainLinks: int(m.GetRelay().GetCertainLinks()),
		},
	}
}

func hopperRules(m *antibotv1.HopperRules) hopper.Rules {
	return hopper.Rules{
		MinAngle:     m.GetMinAngle(),
		MinGap:       m.GetMinGap().AsDuration(),
		MaxGap:       m.GetMaxGap().AsDuration(),
		MinSteps:     int(m.GetMinSteps()),
		MinShare:     m.GetMinShare(),
		CertainSteps: int(m.GetCertainSteps()),
		CertainShare: m.GetCertainShare(),
	}
}

func rotatorRules(m *antibotv1.RotatorRules) rotator.Rules {
	return rotator.Rules{
		Window:       m.GetWindow().AsDuration(),
		MinSeconds:   int(m.GetMinSeconds()),
		MinByChance:  m.GetMinByChance(),
		MaxShare:     m.GetMaxShare(),
		MinSlots:     int(m.GetMinSlots()),
		CertainSlots: int(m.GetCertainSlots()),
	}
}

func durations(of []time.Duration) []*durationpb.Duration {
	out := make([]*durationpb.Duration, 0, len(of))
	for _, d := range of {
		out = append(out, durationpb.New(d))
	}
	return out
}

func durationsOf(of []*durationpb.Duration) []time.Duration {
	out := make([]time.Duration, 0, len(of))
	for _, d := range of {
		out = append(out, d.AsDuration())
	}
	return out
}

func copied(value *float64) *float64 {
	if value == nil {
		return nil
	}
	kept := *value
	return &kept
}

func count(n int) uint32 {
	return uint32(min(max(n, 0), math.MaxUint32)) //nolint:gosec // clamped to the range of uint32 first.
}
