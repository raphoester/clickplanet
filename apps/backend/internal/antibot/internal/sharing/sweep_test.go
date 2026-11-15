package sharing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestTheSweepForgetsTheScopesWhoseTriesAllLapsed(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 11, 15, 18, 0, 0, 0, time.UTC))
	tracker := New(func() Rules { return Rules{Enabled: true, Window: time.Hour, MaxAccounts: 3} }, clock)

	tracker.Tried(detect.Click{Scope: "203.0.113.7", Account: "gone", At: clock.Now()})
	clock.Advance(50 * time.Minute)
	tracker.Tried(detect.Click{Scope: "203.0.113.8", Account: "here", At: clock.Now()})
	clock.Advance(20 * time.Minute)

	tracker.sweep()

	assert.NotContains(t, tracker.scopes, "203.0.113.7")
	assert.Contains(t, tracker.scopes, "203.0.113.8")
}
