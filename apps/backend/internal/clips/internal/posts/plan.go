package posts

import (
	"slices"
	"time"
)

type Plan struct {
	Review    time.Duration
	Spacing   time.Duration
	Platforms []Platform
}

// Each with its time, its platforms and its words: a teaser and a call picked at random, so their variants can be compared.
func (p Plan) Posts(clips []Clip, now time.Time, dice Dice) []Post {
	ranked := slices.SortedFunc(slices.Values(clips), func(a, b Clip) int { return int(a.pick - b.pick) })

	first := now.Add(p.Review).Truncate(time.Minute)
	if first.Before(now.Add(p.Review)) {
		first = first.Add(time.Minute)
	}

	posts := make([]Post, 0, len(ranked))
	for i, clip := range ranked {
		posts = append(posts, Post{
			clip: clip, at: first.Add(time.Duration(i) * p.Spacing), platforms: p.Platforms, wording: wordingOf(clip, dice),
			position: i + 1, ofRun: len(ranked),
		})
	}
	return posts
}
