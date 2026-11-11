package allegiances

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// Lock is true while the rules in force let no account change its flag.
type Lock bool

func (Lock) Validate() error { return nil }

var LockPart = cprules.PartOf[*planetv1.Rules, Lock]{
	Key:  "allegianceLocked",
	Read: func(m *planetv1.Rules) Lock { return Lock(m.GetAllegianceLocked()) },
}
