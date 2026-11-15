//go:build testing

package boxes

import "time"

// TestRules are production's box rules of November 2026, with no welcome.
func TestRules() Rules {
	return Rules{
		Pace:         Pace{MinInterval: 4 * time.Minute, MaxInterval: 8 * time.Minute, MaxBoxesPerHour: 12},
		OfferTTL:     15 * time.Second,
		MissRetry:    2 * time.Minute,
		ActiveWithin: 2 * time.Minute,
	}
}
