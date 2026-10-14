package posts

import (
	"slices"
	"time"
)

type Cadence struct {
	Review  time.Duration
	Spacing time.Duration
}

// Each with its time and its words: a teaser and a call picked at random, so their variants can be compared.
func (c Cadence) Posts(clips []Clip, now time.Time, dice Dice) []Post {
	ranked := slices.SortedFunc(slices.Values(clips), func(a, b Clip) int { return int(a.pick - b.pick) })

	first := now.Add(c.Review).Truncate(time.Minute)
	if first.Before(now.Add(c.Review)) {
		first = first.Add(time.Minute)
	}

	posts := make([]Post, 0, len(ranked))
	for i, clip := range ranked {
		posts = append(posts, Post{clip: clip, at: first.Add(time.Duration(i) * c.Spacing), wording: wordingOf(clip, dice)})
	}
	return posts
}
