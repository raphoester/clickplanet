package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var outcomeCases = []struct {
	name        string
	owner, flag string
	shields     int
	outcome     clicks.Outcome
}{
	{"a tile already wearing the flag is unchanged", "pl", "pl", 0, clicks.Unchanged},
	{"a shielded tile clicked by its own flag is unchanged", "pl", "pl", 3, clicks.Unchanged},
	{"a foreign tile with no shield is taken", "pl", "de", 0, clicks.Taken},
	{"an empty tile is taken", "", "de", 0, clicks.Taken},
	{"a shielded foreign tile keeps its flag", "pl", "de", 1, clicks.Shielded},
	{"many shields shield the same way as one", "pl", "de", 10, clicks.Shielded},
}

func TestOutcomeOf(t *testing.T) {
	for _, c := range outcomeCases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.outcome, clicks.OutcomeOf(c.owner, c.flag, c.shields))
		})
	}
}
