package viruses

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewOutbreak(rules func() charges.Rules, clock cptime.Clock) Outbreak {
	return Outbreak{rules: rules, clock: clock}
}

// Outbreak is the rules in force a virus spreads by, read at each use and tick: how many infections it makes at most,
// and when it spreads next.
type Outbreak struct {
	rules func() charges.Rules
	clock cptime.Clock
}

func (o Outbreak) Most() int {
	return o.rules().Virus.Infections
}

func (o Outbreak) Next() time.Time {
	return o.clock.Now().Add(o.rules().Virus.Every)
}

func (o Outbreak) Due(virus clicks.Virus) bool {
	return virus.Due(o.clock.Now())
}
