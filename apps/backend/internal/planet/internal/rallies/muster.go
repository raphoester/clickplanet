package rallies

import (
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewMuster(rules func() charges.Rules, spacing float64, ids IDProvider, clock cptime.Clock) Muster {
	return Muster{rules: rules, spacing: spacing, ids: ids, clock: clock}
}

// Muster makes a rally by the rules in force as it is called: its ring is charges.rally.rings tile spacings wide, and it
// lasts charges.rally.lasts. A rally keeps both, so a change of the rules reaches only the next one.
type Muster struct {
	rules   func() charges.Rules
	spacing float64
	ids     IDProvider
	clock   cptime.Clock
}

// Radius is a ring's, in arc on the unit globe, by the rules in force.
func (m Muster) Radius() float64 {
	return m.rules().Rally.Rings * m.spacing
}

func (m Muster) Rally(account cpsession.AccountID, flag string, tile uint32) (Rally, error) {
	id, err := m.ids.NewID()
	if err != nil {
		return Rally{}, fmt.Errorf("failed to name the rally: %w", err)
	}

	at := m.clock.Now()
	return Rally{
		id: id, account: account, flag: flag, tile: tile, radius: m.Radius(), calledAt: at,
		endsAt: at.Add(m.rules().Rally.Lasts),
	}, nil
}
