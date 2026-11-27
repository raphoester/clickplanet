package rotator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBurstsThatMissEachOtherInsideASlotAddNoChance(t *testing.T) {
	for name, tc := range map[string]struct {
		turn     func(second int) []string
		spans    int
		byChance float64
	}{
		"minutes apart in the slot": {turn: apart, spans: 0, byChance: 0},
		"in one minute":             {turn: inTurns, spans: 10, byChance: 10 * 40 * 18 / 60.0},
	} {
		t.Run(name, func(t *testing.T) {
			clock := newClock()
			w := newWatchdog(Config{}, bounded(), clock)
			play(w, clock, 10, tc.turn)

			together := w.sharingOf(w.slotOf(clock.Now().Add(-time.Hour)), bounded(), w.accounts["one"], w.accounts["two"])

			assert.Equal(t, tc.spans, together.spans)
			assert.Zero(t, together.seconds)
			assert.InDelta(t, tc.byChance, together.byChance, 1e-9)
		})
	}
}

func TestASpanThatDoesNotDivideTheSlotEndsWithIt(t *testing.T) {
	clock := newClock()
	rules := bounded()
	rules.Span = 280 * time.Second
	w := newWatchdog(Config{}, rules, clock)

	play(w, clock, 1, func(second int) []string {
		switch {
		case second >= 280 && second < 285:
			return []string{"one"}
		case second >= 290 && second < 300:
			return []string{"two"}
		default:
			return nil
		}
	})

	together := w.sharingOf(0, rules, w.accounts["one"], w.accounts["two"])

	assert.Equal(t, 1, together.spans)
	assert.InDelta(t, 5*10/20.0, together.byChance, 1e-9, "the last 20 seconds of the slot are a span")
}

func apart(second int) []string {
	switch {
	case second < 20:
		return []string{"one"}
	case second >= 150 && second < 170:
		return []string{"two"}
	default:
		return nil
	}
}
