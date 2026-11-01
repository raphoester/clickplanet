//go:build testing

package rules

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

// Seeded is the set the migrations keep as default: production's rules on 26 October 2026, and the sign-in refill.
func Seeded() Snapshot {
	d := durationpb.New
	f, n, on := proto.Float64, proto.Uint32, proto.Bool
	return Snapshot{
		Name: Default,
		Rules: &planetv1.Rules{
			Clicks: &planetv1.ClickRules{
				Tempo: f(1), PerSecond: f(0.2), Burst: n(60),
				ScopeMultiplier: f(10), V6ScopeMultiplier: f(4), GuestScopeMultiplier: f(1), LinkedMultiplier: f(2),
				NewAccountClicks: f(10),
			},
			Toll: &planetv1.TollRules{Steps: []*planetv1.TollStep{
				{Share: 0.1, Slowdown: 1.5}, {Share: 0.2, Slowdown: 2.5}, {Share: 0.3, Slowdown: 4},
			}},
			Bonuses: &planetv1.BonusRules{
				Boxes:        &planetv1.BoxPace{MinInterval: d(4 * time.Minute), MaxInterval: d(8 * time.Minute), MaxChargesPerHour: n(12)},
				Kinds:        &planetv1.KindWeights{Refill: f(5), SpreadClicks: f(3), EncloseClicks: f(2), Bomb: f(1), Shields: f(3)},
				OfferTtl:     d(15 * time.Second),
				MissRetry:    d(2 * time.Minute),
				ActiveWithin: d(2 * time.Minute),
				Spread:       &planetv1.SpreadRules{Clicks: n(8), MaxPerBox: n(4)},
				Bomb:         &planetv1.BombRules{Rings: f(4)},
				Enclose:      &planetv1.EncloseRules{MaxTiles: n(25), Held: n(3), MaxPerBox: n(3)},
				Shield:       &planetv1.ShieldRules{Held: n(30), MinPerBox: n(5), MaxPerBox: n(20), PerTile: n(10)},
				Welcome:      &planetv1.WelcomeRules{Within: d(10 * time.Minute), BoxAfter: d(time.Minute)},
				Quiz: &planetv1.QuizRules{
					Enabled: on(true), MinInterval: d(4 * time.Minute), MaxInterval: d(8 * time.Minute),
					MissRetry: d(2 * time.Minute), OfferTtl: d(25 * time.Second), AnswerWindow: d(8 * time.Second),
					LeaderBias: f(1), MaxChargesPerHour: n(12),
				},
				SignInRefill: on(true),
			},
			Frozen:           on(false),
			AllegianceLocked: on(false),
		},
	}
}

func Named(name SetName, change func(*planetv1.Rules)) Snapshot {
	kept := Seeded()
	kept.Name = name
	change(kept.Rules)
	return kept
}

func AsIs(*planetv1.Rules) {}
