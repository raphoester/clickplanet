//go:build testing

package submissions

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func fixed(at time.Time) cptime.Clock {
	return cptime.NewFixedClock(at)
}
