package bonuses

import (
	"fmt"
	"math"
)

// The bounds a rule is checked against, word for word as planet's rules package writes them: bonuses reads no package
// of planet's, and its half of a set must be refused in the same words.

type bound interface {
	~int | ~float64 | ~int64
}

func positive[B bound](name string, value B) error {
	if value > 0 {
		return nil
	}
	return fmt.Errorf("%s is %v, and must be above 0", name, value)
}

func notNegative[B bound](name string, value B) error {
	if value >= 0 {
		return nil
	}
	return fmt.Errorf("%s is %v, and must not be negative", name, value)
}

func atLeast[B bound](name string, value B, floorName string, floor B) error {
	if value >= floor {
		return nil
	}
	return fmt.Errorf("%s is %v, under %s (%v)", name, value, floorName, floor)
}

func finite(name string, value float64) error {
	if !math.IsNaN(value) && !math.IsInf(value, 0) {
		return nil
	}
	return fmt.Errorf("%s is %v, and must be a number", name, value)
}
