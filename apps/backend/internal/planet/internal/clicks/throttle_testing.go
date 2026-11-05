//go:build testing

package clicks

// TestThrottle is a click a second, ten in hand, and every account starting full.
func TestThrottle() Throttle {
	return Throttle{
		Tempo: 1, PerSecond: 1, Burst: 10,
		ScopeMultiplier: 10, V6ScopeMultiplier: 10, GuestScopeMultiplier: 1, LinkedMultiplier: 2,
		NewAccountClicks: 10,
	}
}
