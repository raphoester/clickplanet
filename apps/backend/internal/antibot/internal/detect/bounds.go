package detect

import (
	"fmt"
	"time"
)

type Bound interface {
	~int | ~float64 | time.Duration
}

func Positive[B Bound](name string, value B) error {
	if value > 0 {
		return nil
	}
	return fmt.Errorf("%s is %v, and must be above 0", name, value)
}

func NotNegative[B Bound](name string, value B) error {
	if value >= 0 {
		return nil
	}
	return fmt.Errorf("%s is %v, and must not be negative", name, value)
}

func Share(name string, value float64) error {
	if value >= 0 && value <= 1 {
		return nil
	}
	return fmt.Errorf("%s is %v, and a share is between 0 and 1", name, value)
}

func AtLeast[B Bound](name string, value B, floorName string, floor B) error {
	if value >= floor {
		return nil
	}
	return fmt.Errorf("%s is %v, under %s (%v)", name, value, floorName, floor)
}

func AtMost[B Bound](name string, value B, ceilingName string, ceiling B) error {
	if value <= ceiling {
		return nil
	}
	return fmt.Errorf("%s is %v, over %s (%v)", name, value, ceilingName, ceiling)
}
