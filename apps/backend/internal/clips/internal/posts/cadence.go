package posts

import (
	"slices"
	"time"
)

// When the clips of a run go out: the best one a review from now, each next one a spacing later.
type Cadence struct {
	Review  time.Duration
	Spacing time.Duration
}

func (c Cadence) Posts(clips []Clip, now time.Time) []Post {
	ranked := slices.SortedFunc(slices.Values(clips), func(a, b Clip) int { return int(a.pick - b.pick) })

	first := now.Add(c.Review).Truncate(time.Minute)
	if first.Before(now.Add(c.Review)) {
		first = first.Add(time.Minute)
	}

	posts := make([]Post, 0, len(ranked))
	for i, clip := range ranked {
		posts = append(posts, Post{clip: clip, at: first.Add(time.Duration(i) * c.Spacing)})
	}
	return posts
}
