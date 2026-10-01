package detect_test

import (
	"testing"
	"time"

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

func TestWiderPrefix(t *testing.T) {
	for scope, want := range map[string]string{
		"2a00:8c40:f0c5:6713::/64": "2a00:8c40:f0c0::/44",
		"2a00:8c40:f0ce:fda7::/64": "2a00:8c40:f0c0::/44",
		"203.0.113.7":              "203.0.113.0/24",
		"caller":                   "",
		"10.0.0.0/8":               "",
	} {
		assert.Equal(t, want, detect.WiderPrefix(scope, 24, 44), scope)
	}
}

func TestAReadingWordsTheLevelAndTheEvidenceApart(t *testing.T) {
	at := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	opinion := detect.Opinion{
		Watchdog: "sequencer",
		Verdict:  detect.Suspect,
		Evidence: detect.Evidence{Rule: "constant-stride", Fields: []detect.Field{{Key: "steps", Value: 40}}},
		At:       at,
	}

	assert.Equal(t, detect.Reading{
		Watchdog: "sequencer",
		Level:    "suspect",
		Evidence: "constant-stride steps=40",
		At:       at,
	}, opinion.Reading())

	assert.Equal(t, detect.Reading{Watchdog: "retaker", Level: "clear"}, detect.Opinion{Watchdog: "retaker"}.Reading())
}
