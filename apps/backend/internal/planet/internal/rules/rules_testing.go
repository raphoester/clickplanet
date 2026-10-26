//go:build testing

package rules

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// TestSnapshot is a set that can play, with every value written.
func TestSnapshot(name SetName) Snapshot {
	return Snapshot{
		Name:    name,
		Tempo:   1,
		Clicks:  clicks.TestThrottle(),
		Toll:    []clicks.TollStep{{Share: 0.1, Slowdown: 1.5}, {Share: 0.2, Slowdown: 2.5}},
		Bonuses: bonuses.TestRules(),
	}
}
