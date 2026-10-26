package rules_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

var rushSnapshot = rules.Snapshot{
	Name:  "rush",
	Tempo: 3,
	Boxes: bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30},
}

func TestASetIsWhatItWasMadeOf(t *testing.T) {
	set, err := rushSnapshot.Restore()
	require.NoError(t, err)

	assert.Equal(t, rushSnapshot, set.TakeSnapshot())
}

func TestASetThatCannotPlayIsRefused(t *testing.T) {
	broken := map[string]func(*rules.Snapshot){
		"no name":                   func(s *rules.Snapshot) { s.Name = "" },
		"a name with capitals":      func(s *rules.Snapshot) { s.Name = "Rush" },
		"a name too long":           func(s *rules.Snapshot) { s.Name = "a234567890123456789012345678901234" },
		"no tempo":                  func(s *rules.Snapshot) { s.Tempo = 0 },
		"a tempo that is no number": func(s *rules.Snapshot) { s.Tempo = math.NaN() },
		"no shortest wait":          func(s *rules.Snapshot) { s.Boxes.MinInterval = 0 },
		"a longest wait too short":  func(s *rules.Snapshot) { s.Boxes.MaxInterval = 30 * time.Second },
		"no hourly cap":             func(s *rules.Snapshot) { s.Boxes.MaxChargesPerHour = 0 },
	}
	for name, breaks := range broken {
		t.Run(name, func(t *testing.T) {
			snapshot := rushSnapshot
			breaks(&snapshot)

			_, err := snapshot.Restore()
			require.ErrorIs(t, err, rules.ErrInvalidSet)
		})
	}
}

func TestOneWaitIsAWaitOfItsOwn(t *testing.T) {
	snapshot := rushSnapshot
	snapshot.Boxes.MaxInterval = snapshot.Boxes.MinInterval

	_, err := snapshot.Restore()
	require.NoError(t, err)
}
