package stats

import (
	"encoding/json"
	"time"
)

type PostID string

// A post of the clips job, as the posting service holds it: one clip, with a target on each platform it went to.
type Post struct {
	ID      PostID
	Targets []Target
	// The post's metadata as the service keeps it: its story key, its countries, its call to act and its record.
	Metadata json.RawMessage
}

type Target struct {
	Platform  string
	AccountID string
	// The video's id on the platform, and its address there.
	PlatformPostID string
	URL            string
	// Zero until the platform published it.
	PublishedAt time.Time
}

func (t Target) Out() bool {
	return !t.PublishedAt.IsZero()
}

// Whether any of its targets is out: a post still scheduled has nothing to measure.
func (p Post) Out() bool {
	for _, target := range p.Targets {
		if target.Out() {
			return true
		}
	}
	return false
}
