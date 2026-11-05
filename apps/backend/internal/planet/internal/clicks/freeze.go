package clicks

import (
	"errors"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

var ErrFrozen = errors.New("the map is frozen: it takes no play until the rules change")

// Freeze is true while the rules in force take no play on the map.
type Freeze bool

func (Freeze) Validate() error { return nil }

var FreezePart = rules.PartOf[Freeze]{Key: "frozen", Read: func(m *planetv1.Rules) Freeze { return Freeze(m.GetFrozen()) }}
