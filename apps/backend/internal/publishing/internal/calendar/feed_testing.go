//go:build testing

package calendar

import (
	"slices"
	"time"
)

// With a post at at on the platforms that took it: what the service holds once a post is made.
func (f Feed) With(at time.Time, platforms []Platform) Feed {
	if len(platforms) == 0 {
		return f
	}
	return append(slices.Clone(f), Posting{At: at, Platforms: slices.Clone(platforms)})
}
