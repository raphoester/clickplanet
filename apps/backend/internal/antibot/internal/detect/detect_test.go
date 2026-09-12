package detect_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
)

func TestAQuietWatchdogSaysSoAndNothingElse(t *testing.T) {
	opinion := detect.Opinion{Watchdog: "retaker"}

	assert.False(t, opinion.Fired())
	assert.Equal(t, "clear", opinion.String(),
		"a watchdog that did not fire has no rule and no numbers to report")
}

func TestAReadingNamesItsRuleAndItsNumbers(t *testing.T) {
	opinion := detect.Opinion{
		Watchdog: "sequencer",
		Verdict:  detect.Certain,
		Evidence: detect.Evidence{
			Rule: "constant-stride",
			Fields: []detect.Field{
				{Key: "steps", Value: 200},
				{Key: "share", Value: 0.97},
			},
		},
	}

	assert.True(t, opinion.Fired())
	assert.Equal(t, "certain constant-stride share=0.97 steps=200", opinion.String(),
		"ordered by key, so two lines about the same watchdog read the same way")
}

// The report holds the slice the fields came in on, and the caller is still
// holding the report while it renders it.
func TestRenderingDoesNotReorderTheReportsOwnFields(t *testing.T) {
	fields := []detect.Field{
		{Key: "steps", Value: 200},
		{Key: "share", Value: 0.97},
	}

	opinion := detect.Opinion{
		Verdict:  detect.Suspect,
		Evidence: detect.Evidence{Rule: "constant-stride", Fields: fields},
	}

	_ = opinion.String()

	assert.Equal(t, "steps", fields[0].Key)
	assert.Equal(t, "share", fields[1].Key)
}
