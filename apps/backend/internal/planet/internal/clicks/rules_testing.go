//go:build testing

package clicks

// FixedRules is a rule set in force that changes only when a test changes it.
type FixedRules struct {
	Clicks  Throttle
	Steps   []TollStep
	Shields int
}

func (f *FixedRules) Throttle() Throttle { return f.Clicks }

func (f *FixedRules) Toll() []TollStep { return f.Steps }

func (f *FixedRules) ShieldsPerTile() int { return f.Shields }

// TestThrottle is a click a second, ten in hand, and every account starting full.
func TestThrottle() Throttle {
	return Throttle{
		PerSecond: 1, Burst: 10,
		ScopeMultiplier: 10, V6ScopeMultiplier: 10, GuestScopeMultiplier: 1, LinkedMultiplier: 2,
		NewAccountClicks: 10,
	}
}
