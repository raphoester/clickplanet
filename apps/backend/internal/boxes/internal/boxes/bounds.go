package boxes

import "fmt"

// The bounds a rule is checked against, in the words planet's and the antibot's rule sets refuse a value with.

type bound interface {
	~int | ~int64
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
