package convictions_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/convictions"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
)

func TestAConvictionIsTheReportWithEveryOpinionWorded(t *testing.T) {
	at := time.Date(2026, 10, 31, 21, 15, 0, 0, time.UTC)
	report := detect.Report{
		Scope: "203.0.113.7", Account: "account-b", Alongside: "account-a", Flags: 2, Offence: 1,
		BannedUntil: at.Add(24 * time.Hour),
		Opinions: []detect.Opinion{
			{Watchdog: "retaker"},
			{
				Watchdog: "metronome", Verdict: detect.Suspect, At: at.Add(-time.Minute),
				Evidence: detect.Evidence{Rule: "cadence", Fields: []detect.Field{{Key: "clicks", Value: 120}}},
			},
		},
		Clicks: 120, ActiveFor: time.Hour, LongestGap: time.Second, TopCountry: "fr", TopCountryClicks: 110,
		Tiles: []uint32{1, 2, 3},
	}

	assert.Equal(t, convictions.Conviction{
		At: at, Scope: "203.0.113.7", Account: "account-b", Alongside: "account-a", Flags: 2, Offence: 1,
		BannedUntil: at.Add(24 * time.Hour),
		Readings: []detect.Reading{
			{Watchdog: "retaker", Level: "clear"},
			{Watchdog: "metronome", Level: "suspect", Evidence: "cadence clicks=120", At: at.Add(-time.Minute)},
		},
		Clicks: 120, ActiveFor: time.Hour, LongestGap: time.Second, TopCountry: "fr", TopCountryClicks: 110,
	}, convictions.Of(report, at))
}
