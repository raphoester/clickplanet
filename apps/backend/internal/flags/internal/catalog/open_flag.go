//go:build testing

// Only tests open a flag until the operator's call to open one (step 2 of the regional flags) gives it a caller.
package catalog

import "time"

func OpenFlag(id FlagID, name Name, at time.Time) Flag {
	return Flag{id: id, name: name, state: Open, decidedAt: at}
}
