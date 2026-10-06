package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var outcomeCases = []struct {
	name        string
	owner, flag string
	defenders   int
	outcome     clicks.Outcome
	ownerAfter  string
}{
	{"a tile already wearing the flag is unchanged", "pl", "pl", 0, clicks.Unchanged, "pl"},
	{"a defended tile clicked by its own flag is unchanged", "pl", "pl", 3, clicks.Unchanged, "pl"},
	{"a foreign tile with no defender is taken", "pl", "de", 0, clicks.Taken, "de"},
	{"an empty tile is taken", "", "de", 0, clicks.Taken, "de"},
	{"a defended foreign tile keeps its flag", "pl", "de", 1, clicks.Defended, "pl"},
	{"many defenders defend the same way as one", "pl", "de", 10, clicks.Defended, "pl"},
}

func TestOutcomeOf(t *testing.T) {
	for _, c := range outcomeCases {
		t.Run(c.name, func(t *testing.T) {
			outcome := clicks.OutcomeOf(c.owner, c.flag, c.defenders)
			assert.Equal(t, c.outcome, outcome)
			assert.Equal(t, c.ownerAfter, outcome.OwnerAfter(c.owner, c.flag))
		})
	}
}
