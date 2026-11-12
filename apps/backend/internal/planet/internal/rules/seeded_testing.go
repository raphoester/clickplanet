//go:build testing

package rules

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// Seeded is the set the migrations keep as default: production's rules on 26 October 2026, the sign-in refill and the
// fast start.
func Seeded() cprules.Snapshot[*planetv1.Rules] {
	d := durationpb.New
	f, n, on := proto.Float64, proto.Uint32, proto.Bool
	return cprules.Snapshot[*planetv1.Rules]{
		Name: cprules.Default,
		Rules: &planetv1.Rules{
			Clicks: &planetv1.ClickRules{
				Tempo: f(1), PerSecond: f(0.2), Burst: n(60),
				ScopeMultiplier: f(10), V6ScopeMultiplier: f(4), GuestScopeMultiplier: f(1), LinkedMultiplier: f(2),
				NewAccountClicks: f(10), FastStart: d(30 * time.Minute),
			},
			Toll: &planetv1.TollRules{Steps: []*planetv1.TollStep{
				{Share: 0.1, Slowdown: 1.5}, {Share: 0.2, Slowdown: 2.5}, {Share: 0.3, Slowdown: 4},
			}},
			Charges: &planetv1.ChargeRules{
				Spread:  &planetv1.SpreadRules{Clicks: n(8), Rings: n(1), Unlimited: on(false)},
				Bomb:    &planetv1.BombRules{Rings: f(4), Off: on(false)},
				Enclose: &planetv1.EncloseRules{MaxTiles: n(25), Held: n(3), Unlimited: on(false)},
				Shield:  &planetv1.ShieldRules{Held: n(30), PerTile: n(10)},
			},
			Frozen:           on(false),
			AllegianceLocked: on(false),
			Arena:            &planetv1.ArenaRules{On: on(false), Freeze: d(8 * time.Second)},
			Seed:             &planetv1.Seed{Pattern: &planetv1.Seed_Capitals{Capitals: &planetv1.CapitalSeed{Shields: n(100)}}},
			Reach:            on(false),
			Capitals:         &planetv1.CapitalRules{KnockOut: on(false), FixedShields: on(false)},
		},
	}
}

func Named(name cprules.SetName, change func(*planetv1.Rules)) cprules.Snapshot[*planetv1.Rules] {
	kept := Seeded()
	kept.Name = name
	change(kept.Rules)
	return kept
}

func AsIs(*planetv1.Rules) {}
