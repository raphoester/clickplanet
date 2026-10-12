//go:build testing

package ledger

import (
	"maps"
	"slices"
)

func Kinds() []string { return slices.Sorted(maps.Keys(kinds)) }
