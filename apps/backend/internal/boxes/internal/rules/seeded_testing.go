//go:build testing

package rules

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// Seeded is the set the migrations keep as default on a database with no bonuses rules to copy: production's box half
// of 15 November 2026.
func Seeded() cprules.Snapshot[*boxesv1.Rules] {
	d := durationpb.New
	n, on := proto.Uint32, proto.Bool
	return cprules.Snapshot[*boxesv1.Rules]{
		Name: cprules.Default,
		Rules: &boxesv1.Rules{
			MinInterval:     d(4 * time.Minute),
			MaxInterval:     d(8 * time.Minute),
			MaxBoxesPerHour: n(12),
			OfferTtl:        d(15 * time.Second),
			MissRetry:       d(2 * time.Minute),
			ActiveWithin:    d(2 * time.Minute),
			Welcome:         &boxesv1.WelcomeRules{Within: d(10 * time.Minute), BoxAfter: d(time.Minute)},
			Frozen:          on(false),
		},
	}
}

// Named is default's values under another name, changed: a set that holds every value.
func Named(name cprules.SetName, change func(*boxesv1.Rules)) cprules.Snapshot[*boxesv1.Rules] {
	kept := Seeded()
	kept.Name = name
	change(kept.Rules)
	return kept
}

func AsIs(*boxesv1.Rules) {}
