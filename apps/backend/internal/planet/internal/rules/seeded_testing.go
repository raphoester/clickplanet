//go:build testing

package rules

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

// Seeded is the set the migrations keep as default: production's rules on 26 October 2026, and the sign-in and ad refills.
func Seeded() Snapshot {
	d := durationpb.New
	return Snapshot{
		Name: Default,
		Rules: &planetv1.Rules{
			Clicks: &planetv1.ClickRules{
				Tempo: 1, PerSecond: 0.2, Burst: 60,
				ScopeMultiplier: 10, V6ScopeMultiplier: 4, GuestScopeMultiplier: 1, LinkedMultiplier: 2,
				NewAccountClicks: 10,
			},
			Toll: &planetv1.TollRules{Steps: []*planetv1.TollStep{
				{Share: 0.1, Slowdown: 1.5}, {Share: 0.2, Slowdown: 2.5}, {Share: 0.3, Slowdown: 4},
			}},
			Bonuses: &planetv1.BonusRules{
				Boxes:        &planetv1.BoxPace{MinInterval: d(4 * time.Minute), MaxInterval: d(8 * time.Minute), MaxChargesPerHour: 12},
				Kinds:        &planetv1.KindWeights{Refill: 5, SpreadClicks: 3, EncloseClicks: 2, Bomb: 1, Shields: 3},
				OfferTtl:     d(15 * time.Second),
				MissRetry:    d(2 * time.Minute),
				ActiveWithin: d(2 * time.Minute),
				Spread:       &planetv1.SpreadRules{Clicks: 8, MaxPerBox: 4},
				Bomb:         &planetv1.BombRules{Rings: 4},
				Enclose:      &planetv1.EncloseRules{MaxTiles: 25, Held: 3, MaxPerBox: 3},
				Shield:       &planetv1.ShieldRules{Held: 30, MinPerBox: 5, MaxPerBox: 20, PerTile: 10},
				Welcome:      &planetv1.WelcomeRules{Within: d(10 * time.Minute), BoxAfter: d(time.Minute)},
				Quiz: &planetv1.QuizRules{
					Enabled: true, MinInterval: d(4 * time.Minute), MaxInterval: d(8 * time.Minute),
					MissRetry: d(2 * time.Minute), OfferTtl: d(25 * time.Second), AnswerWindow: d(8 * time.Second),
					LeaderBias: 1, MaxChargesPerHour: 12,
				},
				SignInRefill: true,
				AdRefill:     &planetv1.AdRefillRules{Enabled: true, Every: d(time.Hour)},
			},
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
