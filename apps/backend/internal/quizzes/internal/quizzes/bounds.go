package quizzes

import (
	"fmt"
	"math"
)

// The bounds a rule is checked against, in the words planet's and the antibot's rule sets refuse a value with.

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

func atMost[B bound](name string, value B, ceilingName string, ceiling B) error {
	if value <= ceiling {
		return nil
	}
	return fmt.Errorf("%s is %v, over %s (%v)", name, value, ceilingName, ceiling)
}

func share(name string, value float64) error {
	if value >= 0 && value <= 1 {
		return nil
	}
	return fmt.Errorf("%s is %v, and a share is between 0 and 1", name, value)
}

func finite(name string, value float64) error {
	if !math.IsNaN(value) && !math.IsInf(value, 0) {
		return nil
	}
	return fmt.Errorf("%s is %v, and must be a number", name, value)
}
