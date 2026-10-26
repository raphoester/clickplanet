package rotator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const scope = "2001:db8:1:2::/64"

func bounded() Rules {
	return Rules{Window: 24 * time.Hour, MinSeconds: 15, MinByChance: 90, MaxShare: 0.05, MinSlots: 24}
}

func newWatchdog(config Config, rules Rules, clock cptime.Clock) *Watchdog {
	return New(config, func() Rules { return rules }, clock, func(float64) {})
}

func try(w *Watchdog, clock *cptime.FixedClock, account string) {
	w.Attempted(detect.Click{Scope: scope, Account: account, SignedIn: true, Tile: 1, Country: "hu", At: clock.Now()})
}

func play(w *Watchdog, clock *cptime.FixedClock, slots int, turn func(second int) []string) {
	for range slots {
		for second := range 300 {
			for _, account := range turn(second) {
				try(w, clock, account)
			}
			clock.Advance(time.Second)
		}
	}
}

func inTurns(second int) []string {
	switch {
	case second < 40:
		return []string{"one"}
	case second >= 150 && second < 190:
		return []string{"two"}
	default:
		return nil
	}
}

func sideBySide(second int) []string {
	switch {
	case second < 35:
		return []string{"one"}
	case second < 40:
		return []string{"one", "two"}
	case second < 75:
		return []string{"two"}
	default:
		return nil
	}
}

func newClock() *cptime.FixedClock {
	return cptime.NewFixedClock(time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC))
}

func TestSlotsSurviveASaveAndLoad(t *testing.T) {
	clock := newClock()

	w := newWatchdog(Config{}, bounded(), clock)
	play(w, clock, 30, inTurns)

	data, err := w.Save()
	require.NoError(t, err)

	restarted := newWatchdog(Config{}, bounded(), clock)
	require.NoError(t, restarted.Load(data))

	for _, account := range []string{"one", "two"} {
		next := detect.Click{Scope: scope, Account: account, SignedIn: true, Tile: 1, Country: "hu", At: clock.Now()}
		wantVerdict, wantEvidence := w.Watch(next)
		gotVerdict, gotEvidence := restarted.Watch(next)

		require.Equal(t, detect.Suspect, wantVerdict, account)
		assert.Equal(t, wantVerdict, gotVerdict, account)
		assert.Equal(t, wantEvidence, gotEvidence, account)
	}
}

func TestSlotsSavedUnderAnotherLengthAreDropped(t *testing.T) {
	clock := newClock()

	w := newWatchdog(Config{}, bounded(), clock)
	play(w, clock, 30, inTurns)

	data, err := w.Save()
	require.NoError(t, err)

	restarted := newWatchdog(Config{Slot: 10 * time.Minute}, bounded(), clock)
	require.NoError(t, restarted.Load(data))

	assert.Empty(t, restarted.scopes, "a mark is a second of the slot it was saved under")
}

func TestForgetDropsSlotsBeforeTheCutoff(t *testing.T) {
	clock := newClock()

	w := newWatchdog(Config{}, bounded(), clock)
	play(w, clock, 3, inTurns)

	w.Forget(clock.Now().Add(-5 * time.Minute))
	require.Contains(t, w.scopes, scopeID(scope))
	assert.Len(t, w.scopes[scope].accounts["one"].slots, 1, "the slot the cutoff falls in is kept")

	w.Forget(clock.Now().Add(time.Second))
	assert.Empty(t, w.scopes)
}
