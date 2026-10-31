package allegiances

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

// Lock is true while the rules in force let no account change its flag.
type Lock bool

func (Lock) Validate() error { return nil }

var LockPart = rules.PartOf[Lock]{
	Key:  "allegianceLocked",
	Read: func(m *planetv1.Rules) Lock { return Lock(m.GetAllegianceLocked()) },
}
