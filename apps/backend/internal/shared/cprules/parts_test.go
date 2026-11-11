package cprules_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// The tests play sets of planet.v1.Rules with parts of their own, one of each kind.

type (
	snapshot = cprules.Snapshot[*planetv1.Rules]
	set      = cprules.Set[*planetv1.Rules]
)

// clicks is a part like most: it names each value by its path in the part.
type clicks struct {
	Tempo      float64
	Burst      uint32
	NewAccount float64
}

func (c clicks) Validate() error {
	var reasons []error
	if c.Tempo <= 0 {
		reasons = append(reasons, fmt.Errorf("tempo is %v, and must be above 0", c.Tempo))
	}
	if c.Burst == 0 {
		reasons = append(reasons, fmt.Errorf("burst is %v, and must be above 0", c.Burst))
	}
	if c.NewAccount > float64(c.Burst) {
		reasons = append(reasons, fmt.Errorf("newAccountClicks is %v, over burst (%v)", c.NewAccount, c.Burst))
	}
	return errors.Join(reasons...)
}

var clicksPart = cprules.PartOf[*planetv1.Rules, clicks]{Key: "clicks", Read: func(m *planetv1.Rules) clicks {
	return clicks{
		Tempo: m.GetClicks().GetTempo(), Burst: m.GetClicks().GetBurst(), NewAccount: m.GetClicks().GetNewAccountClicks(),
	}
}}

// shields is a root part, as the offers' is: it names each value by its whole path.
type shields uint32

func (s shields) Validate() error {
	if s == 0 {
		return fmt.Errorf("charges.shield.perTile is %d, and must be above 0", s)
	}
	return nil
}

var shieldsPart = cprules.PartOf[*planetv1.Rules, shields]{Key: "shields", Root: true, Read: func(m *planetv1.Rules) shields {
	return shields(m.GetCharges().GetShield().GetPerTile())
}}

// arena is refused in default alone when it is on, as planet's is.
type arena bool

func (arena) Validate() error { return nil }

var arenaPart = cprules.PartOf[*planetv1.Rules, arena]{
	Key:  "arena",
	Read: func(m *planetv1.Rules) arena { return arena(m.GetArena().GetOn()) },
	InDefault: func(a arena) error {
		if a {
			return errors.New("on: default never plays in an arena")
		}
		return nil
	},
}

type frozen bool

func (frozen) Validate() error { return nil }

var frozenPart = cprules.PartOf[*planetv1.Rules, frozen]{Key: "frozen", Read: func(m *planetv1.Rules) frozen {
	return frozen(m.GetFrozen())
}}

var parts = cprules.Parts[*planetv1.Rules]{clicksPart, shieldsPart, arenaPart, frozenPart}

func seeded() snapshot {
	return snapshot{Name: cprules.Default, Rules: &planetv1.Rules{
		Clicks:  &planetv1.ClickRules{Tempo: proto.Float64(1), Burst: proto.Uint32(60), NewAccountClicks: proto.Float64(10)},
		Charges: &planetv1.ChargeRules{Shield: &planetv1.ShieldRules{PerTile: proto.Uint32(10)}},
		Arena:   &planetv1.ArenaRules{On: proto.Bool(false)},
		Frozen:  proto.Bool(false),
	}}
}

// named is default's values under another name, changed: a set that holds every value.
func named(name cprules.SetName, change func(*planetv1.Rules)) snapshot {
	kept := seeded()
	kept.Name = name
	change(kept.Rules)
	return kept
}

func asIs(*planetv1.Rules) {}

func quicker(r *planetv1.Rules) { r.Clicks.Tempo = proto.Float64(3) }

// rush holds one value of its own, and plays default's for the rest.
func rush() snapshot {
	return snapshot{Name: "rush", Rules: &planetv1.Rules{Clicks: &planetv1.ClickRules{Tempo: proto.Float64(3)}}}
}

func restored(t *testing.T, kept snapshot) set {
	t.Helper()

	restored, err := kept.Restore(parts, seeded())
	require.NoError(t, err)
	return restored
}
