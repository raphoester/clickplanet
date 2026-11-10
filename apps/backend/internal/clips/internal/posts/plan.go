package posts

import "slices"

// What a run asks of the platforms. When each post goes out, and whether a platform has room for it, is the publishing
// module's calendar, which every job that posts keeps.
type Plan struct {
	Platforms []Platform
	// The platforms that take a run's best clip alone.
	BestOnly []Platform
}

// The best first: the order a run's posts go out in.
func Ranked(clips []Clip) []Clip {
	return slices.SortedFunc(slices.Values(clips), func(a, b Clip) int { return int(a.pick - b.pick) })
}

// The platforms the place-th clip of a run goes to, from 1: a platform that takes the best alone takes the first.
func (p Plan) PlatformsOf(place int) []Platform {
	var of []Platform
	for _, platform := range p.Platforms {
		if place == 1 || !slices.Contains(p.BestOnly, platform) {
			of = append(of, platform)
		}
	}
	return of
}

// How many clips a run makes: perRun at most, and no more than the platform with the most room left takes, since a
// clip goes to every platform that still has room; one that takes the best alone takes one. Not the hours: a clip's
// time is known once it is rendered.
func (p Plan) Clips(perRun int, room map[Platform]int) int {
	most := 0
	for _, platform := range p.Platforms {
		left := room[platform]
		if slices.Contains(p.BestOnly, platform) {
			left = min(left, 1)
		}
		most = max(most, left)
	}
	return min(perRun, most)
}

// The post of the clip, the place-th of its run's ofRun, with its words rolled, a teaser and a call picked at random so
// their variants can be compared. Its slot is the calendar's: At.
func (p Plan) Post(clip Clip, place, ofRun int, dice Dice) Post {
	return Post{clip: clip, platforms: p.PlatformsOf(place), wording: wordingOf(clip, dice), position: place, ofRun: ofRun}
}
